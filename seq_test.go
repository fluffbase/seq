package seq

import(
	"testing"
	"fmt"
	//"bufio"
	//"os"
	"io"
)

func TestSeq(t *testing.T) {
	e := Seq{}
	e.Push(Exec{
		Cmd:"echo hello world",
	})
	err := e.Run()
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	output, err := io.ReadAll(e.Pipes.Stdout)
	if err != nil {
		fmt.Printf("error: %v", err)
	}
	if string(output) != "hello world\n" {
		t.Errorf("Echo did not return text: %s", output)
	}
}
