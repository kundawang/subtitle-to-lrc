package converter

import (
	"reflect"
	"strings"
	"testing"

	"github.com/shipurjan/subtitle-to-lrc/converter/shared"
)

func convertVtt(t *testing.T, vttFile string, args shared.UserArgs) ([]string, error) {
	t.Helper()
	lines := strings.Split(vttFile, "\n")
	return ConvertToLyricsFile(lines, "vtt", args)
}

// Karaoke VTT with word-level tags, a repeated cue, a cue without tags,
// and cues crossing minute and hour boundaries.
const karaokeVtt = `WEBVTT

00:00.500 --> 00:04.000
<00:00.500>Hello <00:01.250>beautiful <00:02.000>world

00:04.000 --> 00:07.000
<00:04.000>Hello <00:05.000>beautiful <00:06.000>world

00:07.000 --> 00:09.000
No tags here

00:59.500 --> 01:01.000
<00:59.500>cross <01:00.250>minute

01:00:00.000 --> 01:00:02.000
<01:00:00.500>hour <01:00:01.500>tag`

func TestEnhancedWordLevelTimestamps(t *testing.T) {
	args := shared.UserArgs{Separator: "  ", NoLengthLimit: true, Enhanced: true}
	got, err := convertVtt(t, karaokeVtt, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		"[00:00.50]<00:00.50>Hello <00:01.25>beautiful <00:02.00>world",
		"[00:04.00]<00:04.00>Hello <00:05.00>beautiful <00:06.00>world",
		"[00:07.00]No tags here",
		"[00:09.00]",
		"[00:59.50]<00:59.50>cross <01:00.25>minute",
		"[01:01.00]",
		"[60:00.00]<60:00.50>hour <60:01.50>tag",
		"[60:02.00]",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("enhanced output mismatch:\n got: %q\nwant: %q", got, want)
	}
}

// Multi-line cue with text before the first tag, text after the last tag,
// and formatting tags that must be stripped.
const multilineVtt = `WEBVTT

00:10.000 --> 00:16.000
lead in <00:11.000>first line
second <c>line</c> <00:14.500>tail end`

func TestEnhancedMultilineCue(t *testing.T) {
	args := shared.UserArgs{Separator: "  ", Enhanced: true}
	got, err := convertVtt(t, multilineVtt, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		"[00:10.00]<00:10.00>lead in <00:11.00>first line  second line <00:14.50>tail end",
		"[00:16.00]",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("enhanced output mismatch:\n got: %q\nwant: %q", got, want)
	}
}

const plainVtt = `WEBVTT

00:01.000 --> 00:03.000
first line
second line

00:03.000 --> 00:05.000
next`

func TestPlainModeUnchanged(t *testing.T) {
	args := shared.UserArgs{Separator: "  "}
	got, err := convertVtt(t, plainVtt, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		"[00:01.00]first line  second line",
		"[00:03.00]next",
		"[00:05.00]",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("plain output mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestEnhancedFallsBackToPlainWithoutTags(t *testing.T) {
	args := shared.UserArgs{Separator: "  ", Enhanced: true}
	got, err := convertVtt(t, plainVtt, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		"[00:01.00]first line  second line",
		"[00:03.00]next",
		"[00:05.00]",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("fallback output mismatch:\n got: %q\nwant: %q", got, want)
	}
}

func TestEnhancedRejectsOutOfOrderTags(t *testing.T) {
	vttFile := `WEBVTT

00:00.000 --> 00:05.000
<00:02.000>late <00:01.000>early`

	args := shared.UserArgs{Separator: "  ", Enhanced: true}
	_, err := convertVtt(t, vttFile, args)
	if err == nil || !strings.Contains(err.Error(), "out of order") {
		t.Errorf("expected out-of-order error, got: %v", err)
	}
}

func TestEnhancedRejectsInvalidTagTime(t *testing.T) {
	vttFile := `WEBVTT

00:00.000 --> 00:05.000
<00:75.000>nope`

	args := shared.UserArgs{Separator: "  ", Enhanced: true}
	_, err := convertVtt(t, vttFile, args)
	if err == nil || !strings.Contains(err.Error(), "invalid inline timestamp") {
		t.Errorf("expected invalid timestamp error, got: %v", err)
	}
}

func TestEnhancedLengthLimitStillApplies(t *testing.T) {
	// karaokeVtt ends past the 59:59.99 limit, so without
	// --no-length-limit the conversion must fail.
	args := shared.UserArgs{Separator: "  ", Enhanced: true}
	_, err := convertVtt(t, karaokeVtt, args)
	if err == nil || !strings.Contains(err.Error(), "too long") {
		t.Errorf("expected length limit error, got: %v", err)
	}
}
