package shared

import "time"

// RPC shared types so client and server agree on gob names

type RegisterArgs struct {
	Name string
}

type RegisterReply struct {
	ClientID string
}

type Command struct {
	ClientID      string
	Sequence      uint64
	ReportedX     int
	ReportedY     int
	CommandString string
}

type CommandReply struct {
	Applied bool
	Error   string
}

type GetStateArgs struct {
	ClientID string
}

type PlayerState struct {
	ID   string
	Name string
	X    int
	Y    int
}

type GameState struct {
	Players []PlayerState
	Time    time.Time
}
