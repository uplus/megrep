package cmd

import (
	"os"
	"bufio"
	"fmt"
	"sync"
	"time"
	"regexp"
	"github.com/spf13/cobra"
	"github.com/mattn/go-tty"
	"github.com/morikuni/aec"
)

type Option struct {
	limit int   // output line num per period
	period int  // output limit period(milli seconds)
	stop string // line match this string, stop output
	tail int    // output from tail
}

// RootCmd defines root command
var rootCmd = &cobra.Command{
	Use: "megrep",
	Run: func(cmd *cobra.Command, args []string) { run() },
}

var wg sync.WaitGroup
var option = Option{}

func init() {
	rootCmd.Flags().StringVarP(&option.stop, "stop", "s", "", "stop marker")
	rootCmd.Flags().IntVarP(&option.limit, "limit", "l", 1, "output line num per period")
	rootCmd.Flags().IntVarP(&option.period, "period", "p", 100, "output limit period(milli seconds)")
	// rootCmd.Flags().IntVarP(&option.tail, "tail", "t", 100, "output from tail lines")
}

func Execute() {
	rootCmd.Execute()
}

// Exit finishes a runnning action.
func Exit(err error, codes ...int) {
	var code int
	if len(codes) > 0 {
		code = codes[0]
	} else {
		code = 2
	}
	if err != nil {
		fmt.Println(err)
	}
	os.Exit(code)
}

func run() {
	defer wg.Wait()

	wg.Add(2)
	lineChannel := make(chan string, 10000)
	go scanRoutine(lineChannel)
	go outputRoutine(lineChannel)
}

func scanRoutine(lineChannel chan string) {
	defer wg.Done();

	stdin := bufio.NewScanner(os.Stdin)

	// inr, inw, _ := os.Pipe()
	// orgStdin := os.Stdin
	// inw.Write([]byte(inbuf))
	// inw.Close()
	// os.Stdin = inr
	// fn()
	// os.Stdin = orgStdin

	for stdin.Scan() {
		if err := stdin.Err(); err != nil {
		}

		line := stdin.Text()
		lineChannel <- line
	}

	close(lineChannel)
}

func outputRoutine(lineChannel chan string) {
	defer wg.Done();

	stdout := bufio.NewWriter(os.Stdout)
	ticker := time.NewTicker(time.Duration(option.period) * time.Millisecond)
	stopRegexp := regexp.MustCompile(option.stop)

	// output line if count is not 0
	var count = option.limit

	for {
		if count == 0 {
			// wait interval
			<-ticker.C
			// reset count
			count = option.limit
		} else {
			select {
			case <-ticker.C:
				// reset count
				count = option.limit
			case line, open := <-lineChannel:
				if !open {
					// finish output
					return
				}

				fmt.Fprintln(stdout, line)
				_ = stdout.Flush()
				count--

				if len(option.stop) != 0 && stopRegexp.MatchString(line) {
					// 非同期で読み込みを待って、その間は read(10...) みたいにインクリメントして出力する

					ttyChannel := make(chan string)
					go readTtyAsync(ttyChannel)

					for {
						fmt.Fprint(stdout, "\r")
						select {
						case <-ttyChannel:
							fmt.Fprint(stdout, aec.EraseLine(2))
							goto LEAVE_TTY
						default:
							fmt.Fprintf(stdout, "stop!! please etner to containue (lines %d...)", len(lineChannel))
							_ = stdout.Flush()
							// fmt.Fprintf(stdout, "\rlines %d", )
						}
					}
					LEAVE_TTY:
				}
			}
		}
	}
}

func readTty() string {
	tty, err := tty.Open()
	if err != nil {
		panic("failed to open tty")
	}
	defer tty.Close()

	str, err := tty.ReadPasswordClear()
	if err != nil {
		panic("failed to read from tty")
	}

	return str
}

func readTtyAsync(str chan string) {
	str <- readTty()
}
