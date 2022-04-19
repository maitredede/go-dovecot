package dovecot

import "strings"

type Cmd byte

const (
	// <major> <minor> <value type> <user> <dict name>
	CmdHello Cmd = 'H'

	CmdLookup  Cmd = 'L' // <key>
	CmdIterate Cmd = 'I' // <flags> <path>

	CmdBegin       Cmd = 'B' // <id>
	CmdCommit      Cmd = 'C' // <id>
	CmdCommitAsync Cmd = 'D' // <id>
	CmdRollback    Cmd = 'R' // <id>

	CmdSet       Cmd = 'S' // <id> <key> <value>
	CmdUnset     Cmd = 'U' // <id> <key>
	CmdAtomicInc Cmd = 'A' // <id> <key> <diff>
	CmdTimestamp Cmd = 'T' // <id> <sec> <nsec>
)

type Reply byte

const (
	ReplyError Reply = 0xFF //(-1)

	ReplyOK             Reply = 'O' // <value>
	ReplyMultiOK        Reply = 'M' // protocol v2.2+
	ReplyNotFound       Reply = 'N'
	ReplyFail           Reply = 'F'
	ReplyWriteUncertain Reply = 'W'
	ReplyAsyncCommit    Reply = 'A'
	ReplyIterFinished   Reply = 0x00 // "\0"
	ReplyAsyncID        Reply = '*'
	ReplyAsyncReply     Reply = '+'
)

func Tabescape(unescaped string) string {
	s := unescaped
	s = strings.ReplaceAll(s, "\x01", "\x011")
	s = strings.ReplaceAll(s, "\x00", "\x010")
	s = strings.ReplaceAll(s, "\t", "\x01t")
	s = strings.ReplaceAll(s, "\n", "\x01n")
	s = strings.ReplaceAll(s, "\r", "\x01r")
	return s
}

func Tabunescape(escaped string) string {
	s := escaped
	s = strings.ReplaceAll(s, "\x01r", "\r")
	s = strings.ReplaceAll(s, "\x01n", "\n")
	s = strings.ReplaceAll(s, "\x01t", "\t")
	s = strings.ReplaceAll(s, "\x010", "\x00")
	s = strings.ReplaceAll(s, "\x011", "\x01")
	return s
}

type DataType int

const (
	TypeString DataType = 0
	TypeInt    DataType = 1
)

type IterateFlags int

const (
	IterateFlagRecurse     IterateFlags = 0x01
	IterateFlagSortByKey   IterateFlags = 0x02
	IterateFlagSortByValue IterateFlags = 0x04
	IterateFlagNoValue     IterateFlags = 0x08
	IterateFlagExactKey    IterateFlags = 0x10
	IterateFlagAsync       IterateFlags = 0x20
)

const (
	PathShared  = "shared"
	PathPrivate = "priv"
)

type CommitRet int

const (
	CommitRetOK             CommitRet = 1
	CommitRetNotFound       CommitRet = 0
	CommitRetFailed         CommitRet = -1
	CommitRetWriteUncertain CommitRet = -2
)
