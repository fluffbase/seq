//go:build cgo
package seq

import (
    "fmt"
    "strings"
    "github.com/google/shlex"
    "os/exec"
    "os/user"
    //"bytes"
    //"os"
    "strconv"
    "io"
    //#include <unistd.h>
    //#include <errno.h>
    "C"
)

type Env map[string]string

type Pipes struct {
    Stderr io.ReadCloser
    Stdin io.WriteCloser
    Stdout io.ReadCloser
}
type Cmd interface {
    Run(...Env) (*Pipes, error)
    String(...Env) string
}

type Seq struct {
    Pipes *Pipes
    Vars Env
    Cmds []Cmd
    Next *Seq
    Header string
    Footer string
    Shell string

}

type Exec struct {
    Pipes *Pipes
    Cmd string
    RunAs string
    Sudo bool
}

type Cond struct {
    Pipes *Pipes
    Cmd string
    RunAs string
    Sudo bool
    Do bool
    Else string
}

func (s *Seq) Push(cmds ...Cmd) {
    for _, cmd := range cmds {
        s.Cmds = append(s.Cmds, cmd)
    }
}

func (s *Seq) Append(seq *Seq) {
    sq := s 
    for sq.Next != nil {
        sq = sq.Next
    }
    sq.Next = seq
}

func (s *Seq) Insert(seq *Seq, i int) {
    sq := s
    prev := s
    j := 1
    for j <= i && sq.Next != nil {
        prev = sq
        sq = sq.Next
        j = j + 1
    }

    prev.Next = s
    if sq != prev {
       s.Next = sq
    }
}

func (s *Seq) Run() error {
    _, err := s.run()
    return err
}

func (s *Seq) Attach() (*Pipes, error) {
    return s.run()
}

func (s *Seq) run() (*Pipes, error) {
    if s.Shell == "" {
        s.Shell = "sh -es"
    }
    header := []byte(s.Header)
    args, err := shlex.Split(s.Shell) 
    if err != nil {
        return nil, fmt.Errorf("failed to split args for shell command %s: %v", s.Shell, err)
    }
    sh := exec.Command(args[0], args[1:]...)

    inPipe, err2 := sh.StdinPipe()
    errPipe, err1 := sh.StderrPipe()
    outPipe, err3 := sh.StdoutPipe()
    if err1 != nil || err2 != nil || err3 != nil {
        return nil, fmt.Errorf("Failed to get pipe err: %v\nin: %v\nout: %v", err1, err2, err3)
    }
    s.Pipes = &Pipes {
        Stderr:errPipe,
        Stdin:inPipe,
        Stdout:outPipe,
    }
    sh.Start()

    cursor := s
    inPipe.Write(header)
    for cursor != nil {
        for _, cmd := range s.Cmds {
            inPipe.Write([]byte(cmd.String(s.Vars)+"\n"))
        }
        if s.Next != s {
            cursor = s.Next    
        } else {
            cursor = nil
        }
    }
    inPipe.Write([]byte(s.Footer))
    inPipe.Close()

    return s.Pipes, nil
}

func (v Env) Format(str string) string {
    for k,v := range v {
        str = strings.Replace(str, k, v, -1)
    }
    return str
}

func (e Env) Extend(values map[string]string) {
    for k, v := range values {
        e[k] = v
    }
}

func (e Exec) String(envs ...Env) string {
    cmd := e.Cmd
    if e.Sudo {
        usr := ""
        if e.RunAs != "" {
            usr = fmt.Sprintf("-u %s", e.RunAs)
        }
        cmd = fmt.Sprintf("/usr/bin/sudo %s%s", usr, cmd)
    }
    if len(envs)>0 {
        for _,env := range envs {
            cmd=env.Format(cmd)
        }
    }
    return cmd
}

func (e Exec) Run(envs ...Env) (*Pipes, error) {
    s := Seq{Vars:envs[0], Cmds:[]Cmd{e}}
    if !e.Sudo && e.RunAs != "" {
        u, err := user.Lookup(e.RunAs)
        if err != nil {
            return nil, fmt.Errorf("could not find user %s; %v", e.RunAs, err)
        }
        uid, _ := strconv.ParseInt(u.Uid, 10, 32)
        gid, _ := strconv.ParseInt(u.Gid, 10, 32)
        cerr, errno := C.setgid(C.__gid_t(gid))
        if cerr != 0 {
            return nil, fmt.Errorf("failed to set GID %d due to error %d", gid, errno)
        }
        cerr, errno = C.setuid(C.__uid_t(uid))
        if cerr != 0 {
            return nil, fmt.Errorf("failed to set UID %d due to error %d", uid, errno)
        }
    }
    err := s.Run()
    if err!=nil {
        serr, _ := io.ReadAll(s.Pipes.Stderr)
        return nil, fmt.Errorf("failed to run %s; %v; %s", e.Cmd, err, serr)
    }
    return s.Pipes, nil
}

func (c Cond) String(envs ...Env) string {
    if c.Do {
        return Exec{Cmd:c.Cmd, Sudo:c.Sudo, RunAs:c.RunAs}.String(envs...)
    }
    if c.Else != "" {
        return Exec{Cmd:c.Else, Sudo:c.Sudo, RunAs:c.RunAs}.String(envs...)
    }
    return ":"
}
func (c Cond) Run (env ...Env) (*Pipes, error) {
    if c.Do {
        return Exec{Cmd:c.Cmd, Sudo:c.Sudo, RunAs:c.RunAs}.Run(env...)
    } 
    if c.Else != "" {
        return Exec{Cmd:c.Else, Sudo:c.Sudo, RunAs:c.RunAs}.Run(env...)
    }
    return nil, nil
}
