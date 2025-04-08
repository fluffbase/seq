// seq(uential) is a simple sequential tty wrapper

package seq
import (
    "fmt"
    "strings"
    "github.com/google/shlex"
    "github.com/creack/pty"
    ///"bufio"
    "os/exec"
    "os/user"
    "bytes"
    "os"
    "strconv"
    "io"
)

import (
    //#include <unistd.h>
    //#include <errno.h>
    "C"
)
type Env map[string]string
type Cmd interface {
    Run(...Env) (*os.File, error)
}
type Seq struct {
    Vars Env
    Cmds []Cmd
    Next *Seq
    Pty *os.File
    Reader io.Reader
    Writer io.Writer
}
type Exec struct {
    Cmd string
    Sudo bool
    RunAs string
    
}
type Cond struct {
    Cmd string
    Sudo bool
    Do bool
    Else string
    RunAs string
}
func (s *Seq) Push(cmds ...Cmd) {
    for i,cmd := range cmds {
        fmt.Printf("Adding %dth command: %s\n", i, cmd)
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
func (s *Seq) Attach() (*io.PipeReader, error) {
    return s.run()
}
func (s *Seq) run() (*io.PipeReader, error) {
    r, out := io.Pipe()
    s.Reader = r
    //in, s.Writer := io.Pipe()
    cursor := s
    for cursor != nil {
        for i, cmd := range s.Cmds {
            f, err := cmd.Run(s.Vars)
            if err!=nil {
                return nil, fmt.Errorf("sequence failed running at %d: %v", i, err)
            }
            _, err2 := io.Copy(out, f)
            if err2 != nil {
                return nil, fmt.Errorf("failed copying output writer at %d: %v", i, err2)
            }
           /* _, err := io.Copy(f, in)
            if err != nil {
                return fmt.Errorf("failed copying input writer at %d: %v", i, err)
            }*/
        }
        if s.Next != s {
            cursor = s.Next    
        } else {
            return nil, fmt.Errorf("cyclic sequence detected")
        }
    }   
    out.Close()
    return r, nil
}



func (v Env)Format(str string) string {
    for k,v := range v {
        str = strings.Replace(str, k, v, -1)
    }
    return str
}
func (e Env)Extend(values map[string]string) {
    for k, v := range values {
        e[k] = v
    }
}

func (e Exec) Run(envs ...Env) (*os.File, error) {
    cmd := e.Cmd
    if e.Sudo {
        cmd = fmt.Sprintf("/usr/bin/sudo %s", cmd)
    }
    if len(envs)>0 {
        for _,env := range envs {
            cmd=env.Format(cmd)
        }
    }
    args, err := shlex.Split(cmd)
    if err != nil {
        return nil, fmt.Errorf("failed to split args for %s: %v", cmd, err)
    }
    runnable := exec.Command(args[0], args[1:]...)

    var stderr bytes.Buffer
    runnable.Stderr = &stderr
    if e.RunAs != "" {
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
    f, err := pty.Start(runnable)
    
    if err!=nil {
        return nil, fmt.Errorf("failed to run %s; %v; %s", cmd, err, stderr.String())
    }
    return f, nil
}

func (c Cond) Run (env ...Env) (*os.File, error) {
    if c.Do {
        return Exec{Cmd:c.Cmd, Sudo:c.Sudo, RunAs:c.RunAs}.Run(env...)
    } 
    if c.Else != "" {
        return Exec{Cmd:c.Else, Sudo:c.Sudo, RunAs:c.RunAs}.Run(env...)
    }
    return nil, fmt.Errorf("blank else statement - neither conditional ran")   
}

