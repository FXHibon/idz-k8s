package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	xansi "github.com/charmbracelet/x/ansi"
)

func typeKeys(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		mi, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = asModel(t, mi)
	}
	return m
}

func press(t *testing.T, m Model, k tea.KeyType) Model {
	t.Helper()
	mi, _ := m.handleKey(tea.KeyMsg{Type: k})
	return asModel(t, mi)
}

// TestSearchReopensEmptyWithGhost: reopening '/' after a search starts with
// an empty field (typing a new term needs no erasing) and shows the previous
// term as a ghost that Tab accepts.
func TestSearchReopensEmptyWithGhost(t *testing.T) {
	m := detailModel(t)
	m = typeKeys(t, m, "/image")
	m = press(t, m, tea.KeyEnter)

	m = typeKeys(t, m, "/")
	if !m.searchTyping || m.searchInput != "" {
		t.Fatalf("reopened search must be empty, got %q", m.searchInput)
	}
	if g := m.ghostFor(recallSearch, m.searchInput); g != "image" {
		t.Fatalf("ghost = %q, want the previous term", g)
	}
	view := xansi.Strip(m.View())
	if !strings.Contains(view, "/ ▏image") || !strings.Contains(view, "Tab/→ recall") {
		t.Fatalf("modal must show the ghost after the cursor and the recall hint:\n%s", view)
	}

	m = press(t, m, tea.KeyTab)
	if m.searchInput != "image" {
		t.Fatalf("Tab must accept the ghost, got %q", m.searchInput)
	}
	m = press(t, m, tea.KeyEnter)
	if m.searchQuery != "image" || len(m.searchHits) != 2 {
		t.Fatalf("recalled search must run again (query=%q hits=%v)", m.searchQuery, m.searchHits)
	}
}

// TestSearchGhostSurvivesClearAndCompletesPrefix: Esc clearing the search
// keeps the term recallable; → completes a typed prefix; a diverging input
// drops the ghost and → is left alone.
func TestSearchGhostSurvivesClearAndCompletesPrefix(t *testing.T) {
	m := detailModel(t)
	m = typeKeys(t, m, "/image")
	m = press(t, m, tea.KeyEnter)
	m = press(t, m, tea.KeyEscape)
	if m.searchQuery != "" {
		t.Fatal("Esc must clear the search")
	}

	m = typeKeys(t, m, "/im")
	if g := m.ghostFor(recallSearch, m.searchInput); g != "age" {
		t.Fatalf("prefix ghost = %q, want %q", g, "age")
	}
	m = press(t, m, tea.KeyRight)
	if m.searchInput != "image" {
		t.Fatalf("→ must complete the prefix, got %q", m.searchInput)
	}

	m = typeKeys(t, m, "x")
	if g := m.ghostFor(recallSearch, m.searchInput); g != "" {
		t.Fatalf("diverged input must have no ghost, got %q", g)
	}
	m = press(t, m, tea.KeyTab)
	if m.searchInput != "imagex" {
		t.Fatalf("Tab without a ghost must not touch the field, got %q", m.searchInput)
	}

	m = press(t, m, tea.KeyEscape)
	m = typeKeys(t, m, "/")
	if g := m.ghostFor(recallSearch, m.searchInput); g != "image" {
		t.Fatalf("an Esc-cancelled input must not replace the recalled term, got %q", g)
	}
}

// TestListFilterReopensEmptyWithGhost: same contract on the list filter —
// the active filter becomes the ghost (rows unfiltered while typing) and
// Tab restores it.
func TestListFilterReopensEmptyWithGhost(t *testing.T) {
	m := detailModel(t)
	m.screen = screenList
	m = typeKeys(t, m, "/back")
	m = press(t, m, tea.KeyEnter)
	if m.filter.Value() != "back" {
		t.Fatalf("filter not committed: %q", m.filter.Value())
	}

	m = typeKeys(t, m, "/")
	if !m.filtering || m.filter.Value() != "" {
		t.Fatalf("reopened filter must be empty, got %q", m.filter.Value())
	}
	if !strings.Contains(xansi.Strip(m.View()), "/ ▏back") {
		t.Fatal("list filter modal must show the ghost")
	}
	m = press(t, m, tea.KeyTab)
	if m.filter.Value() != "back" {
		t.Fatalf("Tab must restore the filter, got %q", m.filter.Value())
	}
	m = typeKeys(t, m, " api")
	if m.filter.Value() != "back api" {
		t.Fatalf("typing after recall must append at the end, got %q", m.filter.Value())
	}
}

// TestLiveFiltersReopenEmptyWithGhost: the events, Helm, sizing and usage
// filters follow the same recall contract as lists and content search.
func TestLiveFiltersReopenEmptyWithGhost(t *testing.T) {
	cases := []struct {
		name  string
		sc    screen
		slot  recallSlot
		query func(Model) string
	}{
		{"events", screenEvents, recallEvents, func(m Model) string { return m.eventsQuery }},
		{"helm", screenHelm, recallHelm, func(m Model) string { return m.helmQuery }},
		{"sizing", screenSizingList, recallSizing, func(m Model) string { return m.sizingQuery }},
		{"usage", screenTop, recallUsage, func(m Model) string { return m.usageFilterQ }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := detailModel(t)
			m.screen = c.sc
			m = typeKeys(t, m, "/back")
			m = press(t, m, tea.KeyEnter)
			if c.query(m) != "back" {
				t.Fatalf("filter not committed: %q", c.query(m))
			}
			m = typeKeys(t, m, "/")
			if c.query(m) != "" || m.ghostFor(c.slot, "") != "back" {
				t.Fatalf("reopened filter must be empty with a ghost (query=%q)", c.query(m))
			}
			m = press(t, m, tea.KeyTab)
			m = press(t, m, tea.KeyEnter)
			if c.query(m) != "back" {
				t.Fatalf("Tab must restore the filter, got %q", c.query(m))
			}
		})
	}
}

// TestListFilterCaretFollowsCursor: moving the cursor with ← must move the
// drawn caret too — the caret used to stay at the end while typed text
// landed at the real cursor position.
func TestListFilterCaretFollowsCursor(t *testing.T) {
	m := detailModel(t)
	m.screen = screenList
	m = typeKeys(t, m, "/bck")
	m = press(t, m, tea.KeyLeft)
	m = press(t, m, tea.KeyLeft)
	if v := xansi.Strip(m.View()); !strings.Contains(v, "/ b▏ck") {
		t.Fatalf("caret must be drawn at the cursor (after 'b'):\n%s", v)
	}
	m = typeKeys(t, m, "a")
	if m.filter.Value() != "back" {
		t.Fatalf("typed rune must land at the cursor, got %q", m.filter.Value())
	}
	if v := xansi.Strip(m.View()); !strings.Contains(v, "/ ba▏ck") {
		t.Fatalf("caret must follow the inserted rune:\n%s", v)
	}
}

// TestListFilterRecallOnlyAtEnd: → in the middle of the text moves the
// cursor, it never accepts the ghost.
func TestListFilterRecallOnlyAtEnd(t *testing.T) {
	m := detailModel(t)
	m.screen = screenList
	m = typeKeys(t, m, "/backend")
	m = press(t, m, tea.KeyEnter)
	m = typeKeys(t, m, "/back")
	m = press(t, m, tea.KeyLeft)
	m = press(t, m, tea.KeyRight)
	if m.filter.Value() != "back" {
		t.Fatalf("→ mid-text must only move the cursor, got %q", m.filter.Value())
	}
	m = press(t, m, tea.KeyRight)
	if m.filter.Value() != "backend" {
		t.Fatalf("→ at the end must accept the ghost, got %q", m.filter.Value())
	}
}

// TestInputModalKeepsCaretVisible: a value wider than the modal is cut on
// the left so the caret stays on screen.
func TestInputModalKeepsCaretVisible(t *testing.T) {
	m := detailModel(t)
	long := strings.Repeat("x", 300) + "END"
	out := xansi.Strip(m.inputModal("t", long, -1, "", "h"))
	if !strings.Contains(out, "END▏") || !strings.Contains(out, "/ …") {
		t.Fatalf("caret and the end of the value must stay visible:\n%s", out)
	}
}
