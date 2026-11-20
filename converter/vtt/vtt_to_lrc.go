package vtt

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/shipurjan/subtitle-to-lrc/converter/shared"
)

func ConvertToChunks(vtt_file []string, args shared.UserArgs) ([]shared.SubtitleChunk, error) {
	header_separator_index := slices.Index(vtt_file, "")
	header := vtt_file[:header_separator_index]
	// Header could be used for some metadata in the future
	_ = header

	body := vtt_file[header_separator_index+1:]
	body_chunks, err := SplitBodyIntoChunks(body, args)
	if err != nil {
		return nil, err
	}

	return body_chunks, nil
}

func SplitBodyIntoChunks(body []string, args shared.UserArgs) ([]shared.SubtitleChunk, error) {
	chunks := make([]shared.SubtitleChunk, 0)
	chunk_candidate := make([]string, 0)

	for _, line := range body {
		if line == "" {
			// Get rid of comments etc.
			if IsChunkCandidateValid(chunk_candidate) {
				chunk, err := PostProcessChunk(chunk_candidate, args)
				if err != nil {
					return nil, err
				}
				chunks = append(chunks, chunk)
			}
			chunk_candidate = make([]string, 0)
		} else {
			chunk_candidate = append(chunk_candidate, line)
		}
	}

	// Add the last chunk if EOF is detected
	if len(chunk_candidate) > 0 && IsChunkCandidateValid(chunk_candidate) {
		chunk, err := PostProcessChunk(chunk_candidate, args)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

func IsChunkCandidateValid(chunk_candidate []string) bool {
	for _, line := range chunk_candidate {
		if strings.Contains(line, "-->") {
			return true
		}
	}
	return false
}

func PostProcessChunk(validated_chunk_candidate []string, args shared.UserArgs) (shared.SubtitleChunk, error) {
	var newChunk []string
	for i, line := range validated_chunk_candidate {
		if strings.Contains(line, "-->") {
			newChunk = validated_chunk_candidate[i:]
			break
		}
	}
	start_time, end_time, err := GetChunkTimes(newChunk[0])
	if err != nil {
		return shared.SubtitleChunk{}, err
	}

	chunk := shared.SubtitleChunk{StartTimeMs: start_time, EndTimeMs: end_time, Lines: newChunk[1:]}

	if args.Enhanced {
		// Join the cue lines the same way plain mode does, so word-level
		// output keeps the same text and order as whole-line output.
		joined := strings.Join(chunk.Lines, args.Separator)
		words, err := ParseInlineWords(joined, start_time, end_time)
		if err != nil {
			return shared.SubtitleChunk{}, err
		}
		chunk.Words = words
	}

	return chunk, nil
}

func GetChunkTimes(timeLine string) (int, int, error) {
	if !strings.Contains(timeLine, "-->") {
		return 0, 0, errors.New("invalid time line")
	}
	timestamps := strings.Split(timeLine, "-->")
	start_time, err := TimestampToMilliseconds(strings.Trim(timestamps[0], " "))
	if err != nil {
		return 0, 0, err
	}
	end_time, err := TimestampToMilliseconds(strings.Trim(timestamps[1], " "))
	if err != nil {
		return 0, 0, err
	}

	return start_time, end_time, nil
}

// TimestampToMilliseconds parses a WebVTT timestamp, either mm:ss.ttt
// or hh:mm:ss.ttt, into milliseconds.
func TimestampToMilliseconds(timestamp string) (int, error) {
	parts := strings.Split(timestamp, ":")
	var hours, minutes int
	var secondsPart string
	switch len(parts) {
	case 2:
		minutes, _ = strconv.Atoi(parts[0])
		secondsPart = parts[1]
	case 3:
		hours, _ = strconv.Atoi(parts[0])
		minutes, _ = strconv.Atoi(parts[1])
		secondsPart = parts[2]
	default:
		return 0, fmt.Errorf("invalid timestamp %q", timestamp)
	}
	secondsAndMilliseconds := strings.Split(secondsPart, ".")
	if len(secondsAndMilliseconds) != 2 {
		return 0, fmt.Errorf("invalid timestamp %q", timestamp)
	}
	seconds, _ := strconv.Atoi(secondsAndMilliseconds[0])
	milliseconds, _ := strconv.Atoi(secondsAndMilliseconds[1])

	totalMilliseconds := (hours * 3600000) + (minutes * 60000) + (seconds * 1000) + milliseconds
	return totalMilliseconds, nil
}

// anyTagRegex matches any <...> tag inside a cue payload.
var anyTagRegex = regexp.MustCompile(`<[^>]*>`)

// timestampTagRegex matches a WebVTT timestamp tag:
// either <mm:ss.ttt> or <hh:mm:ss.ttt>.
var timestampTagRegex = regexp.MustCompile(`^<(?:(\d+):)?(\d{1,2}):(\d{2})\.(\d{3})>$`)

// ParseInlineWords splits cue text into words timed by inline WebVTT
// timestamp tags (<mm:ss.ttt> or <hh:mm:ss.ttt>). Tag timestamps are
// absolute (relative to the start of the file), not to the cue.
//
// Degradation and error policy:
//   - text before the first tag is timed at the cue start;
//   - non-timestamp tags (e.g. <c>, <i>, </c>) are formatting and are stripped;
//   - a tag shaped like a timestamp but out of range (e.g. <00:75.000>)
//     is a hard error;
//   - tags must be in non-descending order and not earlier than the cue
//     start, otherwise a hard error is returned;
//   - a tag past the cue end is clamped to the cue end;
//   - empty words (a tag immediately followed by another tag or the end
//     of the text) are skipped;
//   - text without any timestamp tag yields no words, so the caller
//     falls back to plain whole-line output.
func ParseInlineWords(text string, cueStartMs int, cueEndMs int) ([]shared.Word, error) {
	words := make([]shared.Word, 0)
	var buf strings.Builder
	wordStartMs := cueStartMs
	prevTagMs := -1
	sawTimestamp := false

	flush := func() {
		if buf.Len() > 0 {
			words = append(words, shared.Word{TimeMs: wordStartMs, Text: buf.String()})
			buf.Reset()
		}
	}

	pos := 0
	for _, m := range anyTagRegex.FindAllStringIndex(text, -1) {
		buf.WriteString(text[pos:m[0]])
		pos = m[1]

		tag := text[m[0]:m[1]]
		groups := timestampTagRegex.FindStringSubmatch(tag)
		if groups == nil {
			// Not a timestamp tag: treat as formatting and strip it.
			continue
		}

		hours, _ := strconv.Atoi(groups[1])
		minutes, _ := strconv.Atoi(groups[2])
		seconds, _ := strconv.Atoi(groups[3])
		milliseconds, _ := strconv.Atoi(groups[4])
		if (groups[1] != "" && minutes > 59) || seconds > 59 {
			return nil, fmt.Errorf("invalid inline timestamp %s", tag)
		}
		tagMs := hours*3600000 + minutes*60000 + seconds*1000 + milliseconds

		if tagMs < cueStartMs || (prevTagMs >= 0 && tagMs < prevTagMs) {
			return nil, fmt.Errorf("inline timestamp %s is out of order within its cue", tag)
		}
		if tagMs > cueEndMs {
			// Clamp tags that run past the end of the cue.
			tagMs = cueEndMs
		}

		flush()
		wordStartMs = tagMs
		prevTagMs = tagMs
		sawTimestamp = true
	}
	buf.WriteString(text[pos:])
	flush()

	if !sawTimestamp {
		return nil, nil
	}

	return words, nil
}
