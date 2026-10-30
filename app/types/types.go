package types

type Command string

const (
	CommandPing   Command = "PING"
	CommandEcho   Command = "ECHO"
	CommandGet    Command = "GET"
	CommandSet    Command = "SET"
	CommandRPush  Command = "RPUSH"
	CommandLRange Command = "LRANGE"
	CommandLPush  Command = "LPUSH"
	CommandLLen   Command = "LLEN"
	CommandLPop   Command = "LPOP"
	CommandBLpop  Command = "BLPOP"
	CommandType   Command = "TYPE"
	CommandXADD   Command = "XADD"
	CommandXRANGE Command = "XRANGE"
)

type OptionalSetCommandArg string

const (
	OptionalSetCommandArgEX OptionalSetCommandArg = "EX"
	OptionalSetCommandArgPX OptionalSetCommandArg = "PX"
)
