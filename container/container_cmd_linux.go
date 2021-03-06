package container

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/criyle/go-sandbox/pkg/unixsocket"
)

func (c *containerServer) handlePing() error {
	return c.sendReply(&reply{}, nil)
}

func (c *containerServer) handleConf(conf *confCmd) error {
	if conf != nil {
		c.containerConfig = conf.Conf
		if err := initContainer(conf.Conf); err != nil {
			return err
		}
		if c.ContainerUID == 0 {
			c.ContainerUID = containerUID
		}
		if c.ContainerGID == 0 {
			c.ContainerGID = containerGID
		}
	}
	return c.sendReply(&reply{}, nil)
}

func (c *containerServer) handleOpen(open []OpenCmd) error {
	if len(open) == 0 {
		return c.sendErrorReply("open: no open parameter received")
	}

	// open files
	fds := make([]int, 0, len(open))
	for _, o := range open {
		outFile, err := os.OpenFile(o.Path, o.Flag, o.Perm)
		if err != nil {
			return c.sendErrorReply("open: %v", err)
		}
		defer outFile.Close()
		fds = append(fds, int(outFile.Fd()))
	}

	return c.sendReply(&reply{}, &unixsocket.Msg{Fds: fds})
}

func (c *containerServer) handleDelete(delete *deleteCmd) error {
	if delete == nil {
		return c.sendErrorReply("delete: no parameter provided")
	}
	if err := os.Remove(delete.Path); err != nil {
		return c.sendErrorReply("delete: %v", err)
	}
	return c.sendReply(&reply{}, nil)
}

func (c *containerServer) handleReset() error {
	for _, m := range c.Mounts {
		if !m.IsTmpFs() {
			continue
		}
		if err := removeContents(filepath.Join("/", m.Target)); err != nil {
			return c.sendErrorReply("reset: %v %v", m.Target, err)
		}
	}
	return c.sendReply(&reply{}, nil)
}

func (c *containerServer) recvCmd() (*cmd, *unixsocket.Msg, error) {
	cm := new(cmd)
	msg, err := c.socket.RecvMsg(cm)
	if err != nil {
		return nil, nil, err
	}
	return cm, msg, nil
}

func (c *containerServer) sendReply(rep *reply, msg *unixsocket.Msg) error {
	return c.socket.SendMsg(rep, msg)
}

// sendErrorReply sends error reply
func (c *containerServer) sendErrorReply(ft string, v ...interface{}) error {
	errRep := &errorReply{
		Msg: fmt.Sprintf(ft, v...),
	}
	// store errno
	if len(v) == 1 {
		if errno, ok := v[0].(syscall.Errno); ok {
			errRep.Errno = &errno
		}
	}
	return c.sendReply(&reply{Error: errRep}, nil)
}
