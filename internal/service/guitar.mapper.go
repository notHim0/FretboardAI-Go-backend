package service

import "github.com/nothim0/fretboardAI-Go-backend/pkg/python_client"

type GuitarNote struct {
	Time       float64
	Pitch      int
	Duration   float64
	Confidence float64
	Fret       int
	String     int
}

type guitarPosition struct {
	fret   int
	string int
}

// MIDI Pitches for open strings
var openStrings = []int{
	64, //String 1: E4(high E)
	59, // String 2: B3
	55, // String 3: G3
	50, // String 4: D3
	45, // String 5: A2
	40, // String 6: E2 (low E)
}

// MapToGuitar converts raw MIDI notes to guitar fret/string positions
func MapToGuitar(rawNotes []python_client.RawNote) []GuitarNote {
	guitarNotes := make([]GuitarNote, len(rawNotes))

	var prevFret, prevString int

	for i, raw := range rawNotes {
		fret, str := findBestPosition(raw.Pitch, prevFret, prevString)

		guitarNotes[i] = GuitarNote{
			Time:       raw.Time,
			Pitch:      raw.Pitch,
			Duration:   raw.Duration,
			Confidence: raw.Confidence,
			Fret:       fret,
			String:     str,
		}

		prevFret = fret
		prevString = str
	}

	return guitarNotes
}

// findBestPosition determines the best fret/string combination for a MIDI pitch
func findBestPosition(pitch int, prevFret int, prevString int) (int, int) {
	//Find all possible positions for this pitch
	positions := getAllPositions(pitch)

	if len(positions) == 0 {
		return 0, 1
	}

	if prevFret == 0 && prevString == 0 {
		return positions[0].fret, positions[0].string
	}

	//choose position that minimizes hand movement
	bestPos := positions[0]
	bestScore := 1000

	for _, pos := range positions {
		score := calculatePositionScore(pos, prevFret, prevString)

		if score < bestScore {
			bestScore = score
			bestPos = pos
		}
	}
	return bestPos.fret, bestPos.string
}

// getAllPositions finds all the possible fret/string combinations for MIDI pitch
func getAllPositions(pitch int) []guitarPosition {
	var positions []guitarPosition

	for stringNum := 1; stringNum <= 6; stringNum++ {
		openPitch := openStrings[stringNum-1]

		//Calculate fret needed on this string
		fret := pitch - openPitch

		if fret >= 0 && fret <= 24 {
			positions = append(positions, guitarPosition{
				fret:   fret,
				string: stringNum,
			})
		}
	}

	return positions
}

// calculatePositionScore scores a position based on how good it is
// Lower score means better position
func calculatePositionScore(pos guitarPosition, prevFret int, prevString int) int {
	score := 0

	//Factor 1: prefer open strings(fret 0)
	if pos.fret == 0 {
		score -= 20
	}

	//Factor 2: Prefer lower frets(easier to play, brighter tone)
	score += pos.fret * 2

	//Factor 3: Minimize fret distance from previous note (reduce hand movement)
	fretDistance := abs(pos.fret - prevFret)
	score += fretDistance * 5

	//Factor 4: Penalize string changes (staying on same string is smoother)
	if pos.string != prevString {
		score += 10

		//Extra penalty for large string jumps
		stringDistance := abs(pos.string - prevString)
		score += stringDistance * 3
	}

	//Factor 5: Prefer middle strings(2-5) for general playing
	if pos.string == 1 || pos.string == 6 {
		score += 5
	}

	return score
}

// returns absolute value of a integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// GetNoteName converts MIDI pitch to note name
func GetNoteName(pitch int) string {
	noteNames := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	octave := (pitch / 12) - 1
	note := noteNames[pitch%12]

	return note + string(rune('0'+octave))
}
