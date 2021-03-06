package dovecot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/sasha-s/go-deadlock"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type Client interface {
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
	String() string
	User() string
	Dict() string
}

type clientImpl struct {
	h      *DictServer
	conn   net.Conn
	logger *zap.SugaredLogger
	be     Backend

	transactions map[string]*transaction
	txLock       deadlock.Mutex

	major     int
	minor     int
	valueType DataType
	user      string
	dictName  string
}

type transaction struct {
	values map[string]interface{}
}

var _ Client = (*clientImpl)(nil)

func (c *clientImpl) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *clientImpl) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *clientImpl) User() string {
	return c.user
}

func (c *clientImpl) Dict() string {
	return c.dictName
}

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
				c.logger.Debug("client disconnected")
				return
			}
			c.logger.Error("Error reading:", err.Error())
			return
		}

		data := buf[0:reqLen]
		dataStr := string(data)
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

func (c *clientImpl) reply(response Reply, args ...string) error {
	// c.logger.Debugf("replying %v with %v", string([]byte{byte(response)}), args)

	packet := &bytes.Buffer{}
	if _, err := packet.Write([]byte{byte(response)}); err != nil {
		return fmt.Errorf("reply response failed: %w", err)
	}
	argsEscaped := make([]string, len(args))
	for i, a := range args {
		argsEscaped[i] = Tabescape(a)
	}
	argsStr := strings.Join(argsEscaped, "\t")
	if _, err := packet.Write([]byte(argsStr)); err != nil {
		return fmt.Errorf("reply args failed: %w", err)
	}
	if _, err := packet.Write([]byte("\n")); err != nil {
		return fmt.Errorf("reply end failed: %w", err)
	}

	// c.logger.Debugf("packet=%s", packet.String())
	if _, err := packet.WriteTo(c.conn); err != nil {
		return fmt.Errorf("reply send failed: %w", err)
	}
	return nil
}

func (c *clientImpl) processLookup(args []string) error {
	path := args[0]

	c.logger.Debugf("  lookup path=%v", path)

	reply, resultObj, err := c.be.Lookup(c, path)
	if err != nil {
		errReply := c.reply(ReplyError, err.Error())
		return multierr.Combine(err, errReply)
	}

	resultBin, err := json.Marshal(resultObj)
	if err != nil {
		return fmt.Errorf("json marshal failed: %w", err)
	}
	resultString := string(resultBin)
	return c.reply(reply, resultString)
}

func (c *clientImpl) processBegin(args []string) error {
	c.logger.Debugf("processBegin %v", args)

	c.txLock.Lock()
	defer c.txLock.Unlock()

	transactionID := args[0]

	tx := &transaction{
		values: make(map[string]interface{}),
	}
	c.transactions[transactionID] = tx

	return c.reply(ReplyOK, transactionID)
}

func (c *clientImpl) processSet(args []string) error {
	c.logger.Debugf("processSet %v", args)

	c.txLock.Lock()
	defer c.txLock.Unlock()

	transactionID := args[0]
	key := args[1]
	value := args[2]

	tx, ok := c.transactions[transactionID]
	if !ok {
		c.logger.Error("processSet: transaction id=%v does not exists", transactionID)
		return c.reply(ReplyFail, "transaction not found")
	}

	tx.values[key] = value

	return c.reply(ReplyOK)
}

func (c *clientImpl) processNotImplemented(fct string, args []string) error {
	c.logger.Warnf("%v %v", fct, args)
	return c.reply(ReplyFail, "not implemented")
}

func (c *clientImpl) processCommit(args []string) error {
	return c.processNotImplemented("processCommit", args)
}

func (c *clientImpl) processIterate(args []string) error {
	return c.processNotImplemented("processIterate", args)
}

func (c *clientImpl) processCommitAsync(args []string) error {
	return c.processNotImplemented("processCommitAsync", args)
}

func (c *clientImpl) processRollback(args []string) error {
	return c.processNotImplemented("processRollback", args)
}

func (c *clientImpl) processUnset(args []string) error {
	return c.processNotImplemented("processUnset", args)
}

func (c *clientImpl) processAtomicInc(args []string) error {
	return c.processNotImplemented("processAtomicInc", args)
}

func (c *clientImpl) processTimestamp(args []string) error {
	return c.processNotImplemented("processTimestamp", args)
}
