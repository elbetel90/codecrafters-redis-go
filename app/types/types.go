package types

type Command string

const (
	CommandPing Command = "PING"
	CommandEcho Command = "ECHO"
	CommandGet  Command = "GET"
	CommandSet  Command = "SET"
)
