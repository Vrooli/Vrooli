package output

import "testing"

func TestListAcceptsEmptyItems(t *testing.T) {
	List(nil, "")
}

func TestTableAcceptsEmptyRows(t *testing.T) {
	Table([]string{"Name"}, nil)
}
