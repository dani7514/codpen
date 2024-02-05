package crdt

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDocument(t *testing.T) {
	doc := New()

	// A new document must have at least 2 characters (start and end).
	got := doc.Length()
	want := 2

	if got != want {
		t.Errorf("got != want; got = %v, expected = %v\n", got, want)
	}
}

// TestInsert verifies Insert's functionality.
func TestInsert(t *testing.T) {
	doc := New()

	position := 1
	value := "a"

	// Perform insertion.
	content, err := doc.Insert(position, value)
	if err != nil {
		t.Errorf("error: %v\n", err)
	}

	// Generate document for equality assertion.
	wantDoc := &Document{
		Characters: []Character{
			{ID: "start", Visible: false, Value: "", CP: "", CN: "end"},
			{ID: "1", Visible: true, Value: "a", CP: "start", CN: "end"},
			{ID: "end", Visible: false, Value: "", CP: "1", CN: ""},
		},
	}

	got := content
	want := Content(*wantDoc)

	// Since content is a string, it could be compared directly.
	if got != want {
		t.Errorf("got != want; got = %v, expected = %v\n", got, want)
	}
}

// TestIntegrateInsert_SamePosition checks what happens if a value is inserted at the same position.
func TestIntegrateInsert_SamePosition(t *testing.T) {
	// Generate a test document.
	doc := &Document{
		Characters: []Character{
			{ID: "start", Visible: false, Value: "", CP: "", CN: "1"},
			{ID: "1", Visible: false, Value: "e", CP: "start", CN: "2"},
			{ID: "2", Visible: false, Value: "n", CP: "1", CN: "end"},
			{ID: "end", Visible: false, Value: "", CP: "2", CN: ""},
		},
	}

	// Insert a new character at the start. (IDPrevious = start)
	newChar := Character{ID: "3", Visible: false, Value: "b", CP: "start", CN: "1"}

	charPrev := Character{ID: "start", Visible: false, Value: "", CP: "", CN: "1"}
	charNext := Character{ID: "1", Visible: false, Value: "e", CP: "start", CN: "2"}

	// Perform insertion.
	content, err := doc.IntegrateInsert(newChar, charPrev, charNext)
	if err != nil {
		t.Errorf("error: %v\n", err)
	}

	// This should be the final representation of the document.
	wantDoc := &Document{
		Characters: []Character{
			{ID: "start", Visible: false, Value: "", CP: "", CN: "3"},
			{ID: "3", Visible: false, Value: "b", CP: "start", CN: "1"},
			{ID: "1", Visible: false, Value: "e", CP: "3", CN: "2"},
			{ID: "2", Visible: false, Value: "n", CP: "1", CN: "end"},
			{ID: "end", Visible: false, Value: "", CP: "2", CN: ""},
		},
	}

	got := content
	want := wantDoc

	// Do equality check using go-cmp, and display human-readable diff.
	if !cmp.Equal(got, want) {
		t.Errorf("got != want; diff = %v\n", cmp.Diff(got, want))
	}
}

// TestIntegrateInsert_SamePosition checks what happens if a value is inserted at the same position.
func TestIntegrateInsert_BetweenTwoPositions(t *testing.T) {
	// Generate a test document.
	doc := &Document{
		Characters: []Character{
			{ID: "start", Visible: false, Value: "", CP: "", CN: "1"},
			{ID: "1", Visible: false, Value: "c", CP: "start", CN: "2"},
			{ID: "2", Visible: false, Value: "t", CP: "1", CN: "end"},
			{ID: "end", Visible: false, Value: "", CP: "2", CN: ""},
		},
	}

	// Insert a new character between <"1", "c"> and <"2", "t">.
	newChar := Character{ID: "3", Visible: false, Value: "a", CP: "1", CN: "2"}

	charPrev := Character{ID: "1", Visible: false, Value: "c", CP: "start", CN: "2"}
	charNext := Character{ID: "2", Visible: false, Value: "t", CP: "1", CN: "end"}

	// Perform insertion.
	content, err := doc.IntegrateInsert(newChar, charPrev, charNext)
	if err != nil {
		t.Errorf("error: %v\n", err)
	}

	// This should be the final representation of the document.
	wantDoc := &Document{
		Characters: []Character{
			{ID: "start", Visible: false, Value: "", CP: "", CN: "1"},
			{ID: "1", Visible: false, Value: "c", CP: "start", CN: "3"},
			{ID: "3", Visible: false, Value: "a", CP: "1", CN: "2"},
			{ID: "2", Visible: false, Value: "t", CP: "3", CN: "end"},
			{ID: "end", Visible: false, Value: "", CP: "2", CN: ""},
		},
	}

	got := content
	want := wantDoc

	// Do equality check using go-cmp, and display human-readable diff.
	if !cmp.Equal(got, want) {
		t.Errorf("got != want; diff = %v\n", cmp.Diff(got, want))
	}
}

func TestLoad(t *testing.T) {
	// create test doc
	doc := &Document{
		Characters: []Character{
			{ID: "start", Visible: false, Value: "", CP: "", CN: "1"},
			{ID: "1", Visible: true, Value: "c", CP: "start", CN: "3"},
			{ID: "3", Visible: true, Value: "a", CP: "1", CN: "2"},
			{ID: "2", Visible: true, Value: "t", CP: "3", CN: "4"},
			{ID: "4", Visible: true, Value: "\n", CP: "2", CN: "5"},
			{ID: "5", Visible: true, Value: "d", CP: "4", CN: "6"},
			{ID: "6", Visible: true, Value: "o", CP: "5", CN: "7"},
			{ID: "7", Visible: true, Value: "g", CP: "6", CN: "end"},
			{ID: "end", Visible: false, Value: "", CP: "7", CN: ""},
		},
	}

	tmp, err := os.CreateTemp("", "ex")
	if err != nil {
		t.Errorf("error: %v\n", err)
	}
	defer os.Remove(tmp.Name())

	// Save to a temporary file
	err = Save(tmp.Name(), doc)
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}

	// Load from the temporary file
	loadedDoc, err := Load(tmp.Name())
	if err != nil {
		t.Fatalf("error: %v\n", err)
	}
	// compare the contents of the loaded doc and the original doc
	got := Content(loadedDoc)
	want := Content(*doc)

	if !cmp.Equal(got, want) {
		t.Errorf("got != want; diff = %v\n", cmp.Diff(got, want))
	}
}
