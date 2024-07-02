package git

import "log"

type ActionType int

const (
	Insert ActionType = 1
	Update ActionType = 2
	Delete ActionType = 3
)

func (a ActionType) String() string {
	switch a {
	case Insert:
		return "create"
	case Update:
		return "update"
	case Delete:
		return "delete"
	}
	log.Fatalf("invalid git action type: %d", a)
	return ""
}
