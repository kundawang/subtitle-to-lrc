package shared

type UserArgs struct {
	Separator     string
	NoLengthLimit bool
	Enhanced      bool
}

// Word is a single run of lyric text together with the time it appears,
// used for enhanced (word-level) LRC output.
type Word struct {
	TimeMs int
	Text   string
}

type SubtitleChunk struct {
	StartTimeMs int
	EndTimeMs   int
	Lines       []string
	// Words holds word-level timings parsed from inline timestamp tags
	// (e.g. WebVTT karaoke tags). It is empty when the cue has no tags.
	Words []Word
}
