package ui

// Query recall (owner request 2026-11-09): reopening '/' starts with an
// EMPTY field and offers the last committed query as a faint ghost that
// Tab or → accepts. A field pre-filled with the old term had to be erased
// before typing a new one; the ghost keeps both paths one key away.

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type recallSlot int

const (
	recallList recallSlot = iota
	recallSearch
	recallEvents
	recallHelm
	recallSizing
	recallUsage
	recallSlots
)

// openRecall moves the live query into the ghost and empties the field.
func (m *Model) openRecall(slot recallSlot, q *string) {
	m.commitRecall(slot, *q)
	*q = ""
}

// commitRecall remembers a non-empty committed query (Esc never does).
func (m *Model) commitRecall(slot recallSlot, q string) {
	if s := strings.TrimSpace(q); s != "" {
		m.recall[slot] = s
	}
}

// ghostFor returns what remains to type to reach the recalled query, or ""
// when the field already diverged from it.
func (m Model) ghostFor(slot recallSlot, typed string) string {
	last := m.recall[slot]
	if last == "" || typed == last || !strings.HasPrefix(last, typed) {
		return ""
	}
	return last[len(typed):]
}

// acceptRecall completes the field with its ghost on Tab/→. It reports
// false when there is nothing to accept so → keeps its usual meaning.
func (m *Model) acceptRecall(slot recallSlot, msg tea.KeyMsg, q *string) bool {
	if !hit(msg, m.keys.Recall) {
		return false
	}
	g := m.ghostFor(slot, *q)
	if g == "" {
		return false
	}
	*q += g
	return true
}
