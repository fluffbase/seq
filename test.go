package seq

import(
	"testing"
	"fmt"
	"bufio"
	"os"
	"io"
)
func TestSeq(t *testing.T) {

	e := Seq{}
	e.Push(Exec{
		Cmd:"useradd testuser && whoami",
	})
	e.Push(Cond{
		Do:true, 
		RunAs:"testuser", 
		Cmd:"whoami",
	})
	e.Run()
	io.Copy(os.Stdout, e.Reader)
	output := bufio.NewScanner(e.Reader).Text()
	fmt.Printf(output)
}