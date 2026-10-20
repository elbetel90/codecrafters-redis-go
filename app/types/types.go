package types

type Command string

const (
	CommandPing  Command = "PING"
	CommandEcho  Command = "ECHO"
	CommandGet   Command = "GET"
	CommandSet   Command = "SET"
	CommandRPush Command = "RPUSH"
)

type OptionalSetCommandArg string

const (
	OptionalSetCommandArgEX OptionalSetCommandArg = "EX"
	OptionalSetCommandArgPX OptionalSetCommandArg = "PX"
)
