package dovecot

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/sasha-s/go-deadlock"
	"go.uber.org/zap"
)

type Client interface {
}

type clientImpl struct {
	h      *DictServer
	conn   net.Conn
	logger *zap.SugaredLogger
	be     Backend

	major        int
	minor        int
	valueType    DataType
	user         string
	dictName     string
	transactions map[int]interface{}
	txLock       deadlock.Mutex
}

var _ Client = (*clientImpl)(nil)

type commandHandler func(args []string) error

func (c *clientImpl) handleClient() {
	defer c.conn.Close()

	// Make a buffer to hold incoming data.
	buf := make([]byte, 10240)
	for {
		// Read the incoming connection into the buffer.
		reqLen, err := c.conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				c.logger.Warn("client disconnected")
				return
			}
			c.logger.Error("Error reading:", err.Error())
			return
		}

		data := buf[0:reqLen]
		dataStr := string(data)
		c.logger.Debugf("read %v", dataStr)
		lines := strings.Split(dataStr, "\n")
		for _, line := range lines {
			if len(line) < 2 {
				continue
			}
			cmdChar := Cmd(line[0])
			var cmd commandHandler
			//https://github.com/dovecot/core/blob/master/src/lib-dict/dict-client.h
			switch cmdChar {
			case CmdHello:
				cmd = c.processHello
			case CmdLookup:
				cmd = c.processLookup
			case CmdBegin:
				cmd = c.processBegin
			case CmdCommit:
				cmd = c.processCommit
			case CmdSet:
				cmd = c.processSet

			case CmdIterate:
				cmd = c.processIterate
			case CmdCommitAsync:
				cmd = c.processCommitAsync
			case CmdRollback:
				cmd = c.processRollback
			case CmdUnset:
				cmd = c.processUnset
			case CmdAtomicInc:
				cmd = c.processAtomicInc
			case CmdTimestamp:
				cmd = c.processTimestamp
			default:
				c.logger.Warnf("unknown command: %v %v", line[0], string(line[0:0]))
				return
			}

			args := strings.Split(line[1:], "\t")

			if err := cmd(args); err != nil {
				c.logger.Warnf("command '%v' error: %v", line[0], err)
				return
			}
		}

		// // Send a response back to person contacting us.
		// conn.Write([]byte("Message received."))
		// // Close the connection when you're done with it.
		// conn.Close()
	}
}

func (c *clientImpl) processHello(args []string) error {
	major, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("major parse error: %w", err)
	}
	minor, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("minor parse error: %w", err)
	}
	valueType, err := strconv.Atoi(args[2])
	if err != nil {
		return fmt.Errorf("valuetype parse error: %w", err)
	}
	user := args[3]
	dictName := args[4]

	c.major = major
	c.minor = minor
	c.valueType = DataType(valueType)
	c.user = user
	c.dictName = dictName

	c.logger.Debug(c.String())
	return nil
}

func (c *clientImpl) String() string {
	return fmt.Sprintf("client %v.%v type '%v', user '%v', dict '%v'", c.major, c.minor, c.valueType, c.user, c.dictName)
}

func (c *clientImpl) reply(command Reply, args ...string) error {
	c.logger.Debugf("replying %v with %v", command, args)

	if _, err := c.conn.Write([]byte{byte(command)}); err != nil {
		return fmt.Errorf("reply command failed: %w", err)
	}
	argsEscaped := make([]string, len(args))
	for i, v := range args {
		argsEscaped[i] = Tabescape(v)
	}
	argsStr := strings.Join(argsEscaped, "\t")
	if _, err := c.conn.Write([]byte(argsStr)); err != nil {
		return fmt.Errorf("reply args failed: %w", err)
	}
	if _, err := c.conn.Write([]byte("\n")); err != nil {
		return fmt.Errorf("reply end failed: %w", err)
	}
	return nil
}

func (c *clientImpl) processLookup(args []string) error {
	keyArg := args[0]

	c.logger.Debugf("looking up %v", keyArg)
	keyArr := strings.SplitN(keyArg, "/", 2)

	keyType := keyArr[0]
	key := keyArr[1]

	var ns string
	if keyType == "priv" {
		ns = c.user
	}

	c.logger.Debugf("  lookup keyType=%v key=%v ns=%v", keyType, key, ns)

	arr := strings.SplitN(key, "/", 2)
	dictName := arr[0]
	search := arr[1]

	result, err := c.be.Lookup(c, dictName, search)
	if err == nil {
		return c.reply(ReplyOK, result)
	}
	errReply := c.reply(ReplyError, err.Error())
	if errReply != nil {
		return errReply
	}
	return err
}

func (c *clientImpl) processBegin(args []string) error {
	c.logger.Warnf("processBegin %v", args)
	return c.reply(ReplyFail, "not implemented")
}

func (c *clientImpl) processSet(args []string) error {
	c.logger.Warnf("processSet %v", args)
	return c.reply(ReplyFail, "not implemented")
}

func (c *clientImpl) processCommit(args []string) error {
	c.logger.Warnf("processCommit %v", args)
	return c.reply(ReplyFail, "not implemented")
}

func (c *clientImpl) processIterate(args []string) error {
	c.logger.Warnf("processIterate %v", args)
	return c.reply(ReplyFail, "not implemented")
}

func (c *clientImpl) processCommitAsync(args []string) error {
	c.logger.Warnf("processCommitAsync %v", args)
	return c.reply(ReplyFail, "not implemented")
}

func (c *clientImpl) processRollback(args []string) error {
	c.logger.Warnf("processRollback %v", args)
	return c.reply(ReplyFail, "not implemented")
}

func (c *clientImpl) processUnset(args []string) error {
	c.logger.Warnf("processUnset %v", args)
	return c.reply(ReplyFail, "not implemented")
}

func (c *clientImpl) processAtomicInc(args []string) error {
	c.logger.Warnf("processAtomicInc %v", args)
	return c.reply(ReplyFail, "not implemented")
}

func (c *clientImpl) processTimestamp(args []string) error {
	c.logger.Warnf("processTimestamp %v", args)
	return c.reply(ReplyFail, "not implemented")
}
