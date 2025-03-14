package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

var (
	Version string
	Commit  string
	Date    string
)

var rootCmd = &cobra.Command{
	Use:   "cfor [question]",
	Short: "(What's the) command for ...?",
	Long: `Cfor is an AI-powered man page tool that doesn't feel like a thesis.


`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		for {
			fmt.Print("\033[s") // Save cursor position

			s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
			s.Suffix += " "
			s.Color("fgGreen")
			s.Start()

			question := args[0]
			result, err := GenerateCmds(question)
			if err != nil {
				if errors.Is(err, &APIKeyMissingError{}) {
					fmt.Println("\nHave you set up your OpenAI API key? Try one of these:")
					fmt.Println("  export OPENAI_API_KEY=\"sk-...\"")
					fmt.Println("  export CFOR_API_KEY=\"sk-...\"    # For a dedicated key")
				} else if errors.Is(err, &UnsupportedModelError{}) {
					fmt.Println("Unsupported model is specified. Supported models are:")
					fmt.Printf("  %s\n", strings.Join(OpenAISupportedModels, ", "))
				} else {
					fmt.Println("Error generating commands.")
				}

				os.Exit(1)
			}
			s.Stop()

			selectedCmd, err := SelectCmd(result.Message.Cmds)
			if err != nil {
				if errors.Is(err, RerunError{}) {
					fmt.Print("\033[u") // Restore cursor to saved position
					fmt.Print("\033[J") // Clear from cursor to end of screen
					continue
				}

				HandleQuitError(err)
				fmt.Println("Error selecting command")
				os.Exit(1)
			}

			err = injectToPrompt(selectedCmd)
			if err != nil {
				fmt.Println("Error injecting command into prompt")
				os.Exit(1)
			}

			break
		}
	},
}

type InjectError struct{ Char rune }

func (e InjectError) Error() string {
	return fmt.Sprintf("failed to inject character: %c", e.Char)
}

func injectToPrompt(cmd string) error {
	var getTermios, setTermios uint
	var tiocsti, sysIoctl uintptr

	switch runtime.GOOS {
	case "linux":
		getTermios = 0x5401 // unix.TCGETS
		setTermios = 0x5402 // unix.TCSETS
		tiocsti = 0x5412    // syscall.TIOCSTI
		sysIoctl = 16       // syscall.SYS_IOCTL
	case "darwin":
		getTermios = 0x40487413 // unix.TIOCGETA
		setTermios = 0x80487414 // unix.TIOCSETA
		tiocsti = 0x80017472    // syscall.TIOCSTI
		sysIoctl = 54           // syscall.SYS_IOCTL
	}

	// Get the current terminal settings
	termios, err := unix.IoctlGetTermios(int(os.Stdin.Fd()), getTermios)
	if err != nil {
		return fmt.Errorf("failed to get terminal settings: %w", err)
	}

	// Save original settings to restore later
	originalTermios := *termios

	// Disable echo
	termios.Lflag &^= unix.ECHO
	if err := unix.IoctlSetTermios(int(os.Stdin.Fd()), setTermios, termios); err != nil {
		return fmt.Errorf("failed to disable terminal echo: %w", err)
	}

	// Inject the command
	for _, char := range cmd {
		_, _, err := syscall.Syscall(
			sysIoctl,
			os.Stdin.Fd(),
			tiocsti,
			uintptr(unsafe.Pointer(&char)),
		)
		if err != 0 {
			// Restore terminal settings before returning error
			unix.IoctlSetTermios(int(os.Stdin.Fd()), setTermios, &originalTermios)
			return InjectError{Char: char}
		}
	}

	// Restore original terminal settings
	if err := unix.IoctlSetTermios(int(os.Stdin.Fd()), setTermios, &originalTermios); err != nil {
		return fmt.Errorf("failed to restore terminal settings: %w", err)
	}

	return nil
}

func init() {}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
