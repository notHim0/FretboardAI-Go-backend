package llm_client

import (
	"fmt"
	"strings"
)

// systemPrompt tells GPT-4o exactly who it is and what to return
const systemPrompt = `You are an expert guitar music theorist and transcriptionist with decades of experience analyzing guitar performances.

You will be given a list of notes and note groups already mapped to guitar fret/string positions. Each note has:
- time: seconds from start
- pitch: MIDI number
- note_name: human readable (e.g. "E4", "G#3")
- duration: seconds held
- fret: guitar fret (0-24)
- string: guitar string (1=high E, 6=low E)

Note groups show simultaneous or related notes (chords, arpeggios) with their type and note names.

Return ONLY a JSON object with this exact structure, no markdown, no backticks, no explanation:

{
  "key_signature": "string (e.g. E Minor, G Major, A Dorian)",
  "scale_type": "string (e.g. Minor Pentatonic, Natural Minor, Mixolydian)",
  "explanation": "string (2-4 paragraphs: key, scale, chord progression, notable patterns, playing style - written for a guitarist learning this song)",
  "confidence": number between 0 and 1,
  "techniques": [
    {
      "start_time": number,
      "end_time": number,
      "type": "string (hammer_on | pull_off | slide | bend | vibrato | tap | strum | fingerpick | palm_mute | harmonic)",
      "description": "string (e.g. Hammer-on from E to G on 4th string)",
      "confidence": number between 0 and 1,
      "note_indices": [array of integers - which notes from the input this applies to]
    }
  ]
}

Detection rules:
- hammer_on / pull_off: rapid pitch change on same string, no silence gap (< 80ms between notes)
- slide: same string, sequential frets with smooth timing
- bend: note starts slightly below target pitch, glides up
- strum: multiple strings hit within 20ms of each other, downward string order
- fingerpick: multiple strings hit within 20ms, non-sequential string order
- Only include techniques with confidence > 0.6
- Reflect note quality in your overall confidence score`

// buildUserPrompt formats notes and groups into a readable prompt for GPT-4o
func buildUserPrompt(notes []NoteContext, groups []GroupContext) string {
	var sb strings.Builder

	// Notes section
	sb.WriteString(fmt.Sprintf("NOTES (%d total):\n", len(notes)))
	sb.WriteString("idx | time(s) | note  | pitch | fret | string | duration(s)\n")
	sb.WriteString("----|---------|-------|-------|------|--------|----------\n")

	displayNotes := notes
	if len(notes) > 300 {
		displayNotes = sampleNotes(notes, 300)
		sb.WriteString(fmt.Sprintf("(showing 300 of %d notes)\n", len(notes)))
	}

	for i, note := range displayNotes {
		sb.WriteString(fmt.Sprintf("%-3d | %-7.3f | %-5s | %-5d | %-4d | %-6d | %.3f\n",
			i,
			note.Time,
			note.NoteName,
			note.Pitch,
			note.Fret,
			note.String,
			note.Duration,
		))
	}

	// Note groups section
	if len(groups) > 0 {
		sb.WriteString(fmt.Sprintf("\nNOTE GROUPS (%d total):\n", len(groups)))
		sb.WriteString("time(s) | type      | notes              | duration(s)\n")
		sb.WriteString("--------|-----------|--------------------|-----------\n")

		for _, group := range groups {
			noteList := strings.Join(group.NoteNames, ", ")
			sb.WriteString(fmt.Sprintf("%-7.3f | %-9s | %-18s | %.3f\n",
				group.Time,
				group.GroupType,
				noteList,
				group.Duration,
			))
		}
	}

	sb.WriteString("\nAnalyze these notes and return the JSON.")
	return sb.String()
}

// sampleNotes evenly samples n notes from a larger slice
func sampleNotes(notes []NoteContext, n int) []NoteContext {
	if len(notes) <= n {
		return notes
	}
	sampled := make([]NoteContext, n)
	step := float64(len(notes)) / float64(n)
	for i := 0; i < n; i++ {
		idx := int(float64(i) * step)
		sampled[i] = notes[idx]
	}
	return sampled
}

// midiToNoteName converts a MIDI note number to human-readable name
// e.g. 64 -> "E4", 69 -> "A4"
func midiToNoteName(pitch int) string {
	noteNames := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	octave := (pitch / 12) - 1
	note := noteNames[pitch%12]
	return fmt.Sprintf("%s%d", note, octave)
}

// BuildNoteContexts converts NoteInput slice into NoteContext slice for the prompt
func BuildNoteContexts(inputs []NoteInput) []NoteContext {
	contexts := make([]NoteContext, len(inputs))
	for i, n := range inputs {
		contexts[i] = NoteContext{
			Time:     n.Time,
			Pitch:    n.Pitch,
			NoteName: midiToNoteName(n.Pitch),
			Duration: n.Duration,
			Fret:     n.Fret,
			String:   n.String,
		}
	}
	return contexts
}

// BuildGroupContexts converts GroupInput slice into GroupContext slice for the prompt
func BuildGroupContexts(inputs []GroupInput) []GroupContext {
	contexts := make([]GroupContext, len(inputs))
	for i, g := range inputs {
		contexts[i] = GroupContext{
			Time:      g.Time,
			GroupType: g.GroupType,
			NoteNames: g.NoteNames,
			Duration:  g.Duration,
		}
	}
	return contexts
}