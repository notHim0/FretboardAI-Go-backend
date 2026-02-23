package service

import (
	"fmt"
	"sort"
	"strings"
)

// GuitarNoteGroup represents a collection of notes (chord, arpeggio, etc.)
type GuitarNoteGroup struct {
	Time       float64
	Duration   float64
	GroupType  string // "chord", "arpeggio", "double_stop", "power_chord"
	Name       string // "Em", "G Major", "E5"
	Notes      []GuitarNote
	NoteNames  []string
	Confidence float64
}

func DetectChords(notes []GuitarNote, timeThreshold float64) []GuitarNoteGroup {
	if len(notes) == 0 {
		return []GuitarNoteGroup{}
	}

	var groups []GuitarNoteGroup
	timeThresholdSec := timeThreshold / 1000.0

	//sort notes by time
	sortedNotes := make([]GuitarNote, len(notes))
	copy(sortedNotes, notes)
	sort.Slice(sortedNotes, func(i, j int) bool {
		return sortedNotes[i].Time < sortedNotes[j].Time
	})

	//Group notes that occur within the time threshold
	currentGroup := []GuitarNote{sortedNotes[0]}

	for i := 1; i < len(sortedNotes); i++ {
		timeDiff := sortedNotes[i].Time - currentGroup[0].Time

		if timeDiff <= timeThresholdSec {
			currentGroup = append(currentGroup, sortedNotes[i])
		} else {
			if len(currentGroup) >= 2 {
				group := createNoteGroup(currentGroup)
				groups = append(groups, group)
			}

			currentGroup = []GuitarNote{sortedNotes[i]}
		}
	}

	if len(currentGroup) >= 2 {
		group := createNoteGroup(currentGroup)
		groups = append(groups, group)
	}

	return groups
}

func createNoteGroup(notes []GuitarNote) GuitarNoteGroup {
	pitches := make(map[int]bool)
	for _, n := range notes {
		pitches[n.Pitch] = true
	}

	uniquePitches := make([]int, 0, len(pitches))
	for p := range pitches {
		uniquePitches = append(uniquePitches, p)
	}
	sort.Ints(uniquePitches)

	//convert to note names
	noteNames := make([]string, len(uniquePitches))
	for i, p := range uniquePitches {
		noteNames[i] = GetNoteName(p)
	}

	totalTime := 0.0
	maxDuraion := 0.0
	totalConfidence := 0.0

	for _, n := range notes {
		totalTime += n.Time
		if n.Duration > maxDuraion {
			maxDuraion = n.Duration
		}
		totalConfidence += n.Confidence
	}

	avgTime := totalTime / float64(len(notes))
	avgConfidence := totalConfidence / float64(len(notes))

	groupType, name := identifyChord(uniquePitches, len(notes))

	return GuitarNoteGroup{
		Time:       avgTime,
		Duration:   maxDuraion,
		GroupType:  groupType,
		Name:       name,
		Notes:      notes,
		NoteNames:  noteNames,
		Confidence: avgConfidence,
	}
}

func identifyChord(pitches []int, noteCount int) (string, string) {
	if len(pitches) == 0 {
		return "unknown", "Unknown"
	}

	intervals := make([]int, len(pitches))
	root := pitches[0] % 12

	for i, p := range pitches {
		intervals[i] = (p - pitches[0]) % 12
	}

	sort.Ints(intervals)

	rootName := getNoteName(root)

	//special case: Power chrod (root + perfect 5th)
	if len(intervals) == 2 && intervals[1] == 7 {
		return "power_chord", rootName + "5"
	}

	if len(intervals) == 2 {
		return "double_stop", fmt.Sprintf("%s/%s", getNoteName(pitches[0]%12), getNoteName(pitches[1]%12))
	}

	//Chord detection based on intervals
	// Chord detection based on intervals
	if len(intervals) >= 3 {
		// Major chord: root, major 3rd (4), perfect 5th (7)
		if contains(intervals, 0) && contains(intervals, 4) && contains(intervals, 7) {
			return "chord", rootName + " Major"
		}

		// Minor chord: root, minor 3rd (3), perfect 5th (7)
		if contains(intervals, 0) && contains(intervals, 3) && contains(intervals, 7) {
			return "chord", rootName + " Minor"
		}

		// Dominant 7th: root, major 3rd (4), perfect 5th (7), minor 7th (10)
		if contains(intervals, 0) && contains(intervals, 4) && contains(intervals, 7) && contains(intervals, 10) {
			return "chord", rootName + "7"
		}

		// Major 7th: root, major 3rd (4), perfect 5th (7), major 7th (11)
		if contains(intervals, 0) && contains(intervals, 4) && contains(intervals, 7) && contains(intervals, 11) {
			return "chord", rootName + "maj7"
		}

		// Minor 7th: root, minor 3rd (3), perfect 5th (7), minor 7th (10)
		if contains(intervals, 0) && contains(intervals, 3) && contains(intervals, 7) && contains(intervals, 10) {
			return "chord", rootName + "m7"
		}

		// Suspended 2nd: root, major 2nd (2), perfect 5th (7)
		if contains(intervals, 0) && contains(intervals, 2) && contains(intervals, 7) {
			return "chord", rootName + "sus2"
		}

		// Suspended 4th: root, perfect 4th (5), perfect 5th (7)
		if contains(intervals, 0) && contains(intervals, 5) && contains(intervals, 7) {
			return "chord", rootName + "sus4"
		}

		// Diminished: root, minor 3rd (3), diminished 5th (6)
		if contains(intervals, 0) && contains(intervals, 3) && contains(intervals, 6) {
			return "chord", rootName + "dim"
		}

		// Augmented: root, major 3rd (4), augmented 5th (8)
		if contains(intervals, 0) && contains(intervals, 4) && contains(intervals, 8) {
			return "chord", rootName + "aug"
		}
	}

	if noteCount >= 3 {
		return "arpeggio", rootName + " Arpeggio"
	}

	return "chord", rootName + " Chord"
}

// getNoteName returns the note name for a pitch class (0-11)
func getNoteName(pitchClass int) string {
	noteNames := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	return noteNames[pitchClass%12]
}

// contains checks if a slice contains a value
func contains(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}

	return false
}

// GetChordNotes returns a human-readable list of note names in a chord
func GetChordNotes(pitches []int) string {
	names := make([]string, len(pitches))
	for i, p := range pitches {
		names[i] = GetNoteName(p)
	}
	return strings.Join(names, ", ")
}
