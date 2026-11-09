package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	xansi "github.com/charmbracelet/x/ansi"
)

func typeKeys(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		mi, _ := m.handleKey(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = asModel(t, mi)
	}
	return m
}

// aroundCursor returns the cells left of and under the real terminal cursor.
func aroundCursor(t *testing.T, m Model) (prev, at rune) {
	t.Helper()
	v := m.View()
	if v.Cursor == nil {
		t.Fatal("an open input must show the terminal cursor")
	}
	lines := strings.Split(xansi.Strip(v.Content), "\n")
	line := []rune(lines[v.Cursor.Y])
	return line[v.Cursor.X-1], line[v.Cursor.X]
}

func press(t *testing.T, m Model, k rune) Model {
	t.Helper()
	mi, _ := m.handleKey(tea.KeyPressMsg{Code: k})
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
	view := xansi.Strip(m.View().Content)
	if !strings.Contains(view, "/ image") || !strings.Contains(view, "Tab/→ recall") {
		t.Fatalf("modal must show the ghost and the recall hint:\n%s", view)
	}
	if prev, at := aroundCursor(t, m); prev != ' ' || at != 'i' {
		t.Fatalf("cursor must sit right before the ghost, got %q|%q", prev, at)
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
	if !strings.Contains(xansi.Strip(m.View().Content), "/ back") {
		t.Fatal("list filter modal must show the ghost")
	}
	if _, at := aroundCursor(t, m); at != 'b' {
		t.Fatalf("cursor must sit on the ghost's first rune, got %q", at)
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

// TestListFilterCaretFollowsCursor: the terminal cursor follows ←/→ and
// insertion, and nothing is drawn for it — a drawn caret either stayed at
// the end or took a column and shifted the text right.
func TestListFilterCaretFollowsCursor(t *testing.T) {
	m := detailModel(t)
	m.screen = screenList
	m = typeKeys(t, m, "/bck")
	m = press(t, m, tea.KeyLeft)
	m = press(t, m, tea.KeyLeft)
	if v := xansi.Strip(m.View().Content); !strings.Contains(v, "/ bck ") || strings.Contains(v, "▏") {
		t.Fatalf("the text must stay unshifted, with no drawn caret:\n%s", v)
	}
	if prev, at := aroundCursor(t, m); prev != 'b' || at != 'c' {
		t.Fatalf("cursor must sit between 'b' and 'c', got %q|%q", prev, at)
	}
	m = typeKeys(t, m, "a")
	if m.filter.Value() != "back" {
		t.Fatalf("typed rune must land at the cursor, got %q", m.filter.Value())
	}
	if prev, at := aroundCursor(t, m); prev != 'a' || at != 'c' {
		t.Fatalf("cursor must follow the inserted rune, got %q|%q", prev, at)
	}
	m = press(t, m, tea.KeyEnd)
	if prev, at := aroundCursor(t, m); prev != 'k' || at != ' ' {
		t.Fatalf("End must put the cursor after the text, got %q|%q", prev, at)
	}
}

// TestCursorIsABarAndOnlyWhileTyping: the real cursor is a bar, shown only
// while an input is open.
func TestCursorIsABarAndOnlyWhileTyping(t *testing.T) {
	m := detailModel(t)
	m.screen = screenList
	if m.View().Cursor != nil {
		t.Fatal("no input open: the cursor must stay hidden")
	}
	m = typeKeys(t, m, "/")
	if c := m.View().Cursor; c == nil || c.Shape != tea.CursorBar {
		t.Fatalf("open input: want a bar cursor, got %+v", c)
	}
	m = press(t, m, tea.KeyEscape)
	if m.View().Cursor != nil {
		t.Fatal("closed input: the cursor must hide again")
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
// the left so the caret stays inside the box.
func TestInputModalKeepsCaretVisible(t *testing.T) {
	m := detailModel(t)
	long := strings.Repeat("x", 300) + "END"
	box, cx := m.inputModal("t", long, -1, "", "h")
	field := []rune(strings.Split(xansi.Strip(box), "\n")[inputCaretRow])
	if !strings.Contains(string(field), "/ …") || string(field[cx-3:cx]) != "END" {
		t.Fatalf("caret (col %d) must follow the end of the value:\n%s", cx, xansi.Strip(box))
	}
}

// TestPickerQueryShowsCursor: the ':' picker's type-to-filter line gets the
// same real cursor as the input boxes.
func TestPickerQueryShowsCursor(t *testing.T) {
	m := detailModel(t)
	m.screen = screenList
	m = typeKeys(t, m, ":dep")
	if m.screen != screenPicker {
		t.Fatal("':' must open the picker")
	}
	if prev, at := aroundCursor(t, m); prev != 'p' || at != ' ' {
		t.Fatalf("cursor must follow the picker query, got %q|%q", prev, at)
	}
}

// TestCursorAccountsForBanner: the kikoo banner is prepended above the
// overlays, so the cursor row must shift with it.
func TestCursorAccountsForBanner(t *testing.T) {
	m := detailModel(t)
	m.kikoo = true
	m.width, m.height = 120, 40
	m.layout()
	if m.bannerH() == 0 {
		t.Fatal("test setup: the banner must be visible")
	}
	m.screen = screenList
	m = typeKeys(t, m, "/ab")
	if prev, at := aroundCursor(t, m); prev != 'b' || at != ' ' {
		t.Fatalf("cursor row must include the banner offset, got %q|%q", prev, at)
	}
}
