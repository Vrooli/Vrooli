package gates

import (
	"reflect"
	"testing"
)

func TestInteractiveElementsRespectsQuotedAndExpressionDelimiters(t *testing.T) {
	source := `<div><button title=">" data-value={condition ? "<x>" : "y"}>Save</button><a href='/x'>open</a><CustomButton/><input value={items.map((item) => item.label)} /></div>`
	got := interactiveElements(source)
	want := []string{
		`<button title=">" data-value={condition ? "<x>" : "y"}>`,
		`<a href='/x'>`,
		`<input value={items.map((item) => item.label)} />`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("interactiveElements() = %#v, want %#v", got, want)
	}
}

func TestInteractiveElementsReturnsEmptyWithoutInteractiveTags(t *testing.T) {
	if got := interactiveElements(`<main><div>plain content</div><Button /></main>`); len(got) != 0 {
		t.Fatalf("interactiveElements() = %#v, want no native interactive elements", got)
	}
}
