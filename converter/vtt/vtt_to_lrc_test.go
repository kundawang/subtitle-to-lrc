package vtt

import (
	"reflect"
	"strings"
	"testing"

	"github.com/shipurjan/subtitle-to-lrc/converter/shared"
)

func TestParseInlineWordsClampsTagPastCueEnd(t *testing.T) {
	got, err := ParseInlineWords("<00:01.000>a <00:05.000>b", 0, 3000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []shared.Word{
		{TimeMs: 1000, Text: "a "},
		{TimeMs: 3000, Text: "b"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseInlineWordsSkipsEmptyWords(t *testing.T) {
	got, err := ParseInlineWords("<00:01.000><00:02.000>word", 0, 5000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []shared.Word{
		{TimeMs: 2000, Text: "word"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseInlineWordsNoTagsReturnsNil(t *testing.T) {
	got, err := ParseInlineWords("plain text", 0, 5000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil words for tagless text, got %+v", got)
	}
}

func TestParseInlineWordsHourForm(t *testing.T) {
	got, err := ParseInlineWords("<01:02:03.500>x", 0, 4000000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []shared.Word{
		{TimeMs: 3723500, Text: "x"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseInlineWordsTextAroundTagsKeepsOrder(t *testing.T) {
	got, err := ParseInlineWords("before <00:01.000>middle <c.class>still</c> middle <00:02.000>after", 0, 5000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []shared.Word{
		{TimeMs: 0, Text: "before "},
		{TimeMs: 1000, Text: "middle still middle "},
		{TimeMs: 2000, Text: "after"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestParseInlineWordsOutOfOrder(t *testing.T) {
	_, err := ParseInlineWords("<00:02.000>a <00:01.000>b", 0, 5000)
	if err == nil || !strings.Contains(err.Error(), "out of order") {
		t.Errorf("expected out-of-order error, got: %v", err)
	}
}

func TestParseInlineWordsTagBeforeCueStart(t *testing.T) {
	_, err := ParseInlineWords("<00:01.000>a", 2000, 5000)
	if err == nil || !strings.Contains(err.Error(), "out of order") {
		t.Errorf("expected out-of-order error, got: %v", err)
	}
}

func TestParseInlineWordsInvalidTime(t *testing.T) {
	for _, text := range []string{"<00:75.000>a", "<01:99:00.000>a"} {
		_, err := ParseInlineWords(text, 0, 5000000)
		if err == nil || !strings.Contains(err.Error(), "invalid inline timestamp") {
			t.Errorf("text %q: expected invalid timestamp error, got: %v", text, err)
		}
	}
}
