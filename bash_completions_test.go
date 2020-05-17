package cobra

import (
	"bytes"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

func checkOmit(t *testing.T, found, unexpected string) {
	if strings.Contains(found, unexpected) {
		t.Errorf("Got: %q\nBut should not have!\n", unexpected)
	}
}

func check(t *testing.T, found, expected string) {
	if !strings.Contains(found, expected) {
		t.Errorf("Expecting to contain: \n %q\nGot:\n %q\n", expected, found)
	}
}

func checkNumOccurrences(t *testing.T, found, expected string, expectedOccurrences int) {
	numOccurrences := strings.Count(found, expected)
	if numOccurrences != expectedOccurrences {
		t.Errorf("Expecting to contain %d occurrences of: \n %q\nGot %d:\n %q\n", expectedOccurrences, expected, numOccurrences, found)
	}
}

func checkRegex(t *testing.T, found, pattern string) {
	matched, err := regexp.MatchString(pattern, found)
	if err != nil {
		t.Errorf("Error thrown performing MatchString: \n %s\n", err)
	}
	if !matched {
		t.Errorf("Expecting to match: \n %q\nGot:\n %q\n", pattern, found)
	}
}

func runShellCheck(s string) error {
	excluded := []string{
		"SC2034", // PREFIX appears unused. Verify it or export it.
	}
	cmd := exec.Command("shellcheck", "-s", "bash", "-", "-e", strings.Join(excluded, ","))
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	go func() {
		stdin.Write([]byte(s))
		stdin.Close()
	}()

	return cmd.Run()
}

// World worst custom function, just keep telling you to enter hello!
const bashCompletionFunc = `__root_custom_func() {
	COMPREPLY=( "hello" )
}
`

func TestBashCompletions(t *testing.T) {
	rootCmd := &Command{
		Use:                    "root",
		ArgAliases:             []string{"pods", "nodes", "services", "replicationcontrollers", "po", "no", "svc", "rc"},
		ValidArgs:              []string{"pod", "node", "service", "replicationcontroller"},
		BashCompletionFunction: bashCompletionFunc,
		Run:                    emptyRun,
	}
	rootCmd.Flags().IntP("introot", "i", -1, "help message for flag introot")
	rootCmd.MarkFlagRequired("introot")

	// Filename.
	rootCmd.Flags().String("filename", "", "Enter a filename")
	rootCmd.MarkFlagFilename("filename", "json", "yaml", "yml")

	// Persistent filename.
	rootCmd.PersistentFlags().String("persistent-filename", "", "Enter a filename")
	rootCmd.MarkPersistentFlagFilename("persistent-filename")
	rootCmd.MarkPersistentFlagRequired("persistent-filename")

	// Filename extensions.
	rootCmd.Flags().String("filename-ext", "", "Enter a filename (extension limited)")
	rootCmd.MarkFlagFilename("filename-ext")
	rootCmd.Flags().String("custom", "", "Enter a filename (extension limited)")
	rootCmd.MarkFlagCustom("custom", "__complete_custom")

	// Subdirectories in a given directory.
	rootCmd.Flags().String("theme", "", "theme to use (located in /themes/THEMENAME/)")
	rootCmd.Flags().SetAnnotation("theme", BashCompSubdirsInDir, []string{"themes"})

	// For two word flags check
	rootCmd.Flags().StringP("two", "t", "", "this is two word flags")
	rootCmd.Flags().BoolP("two-w-default", "T", false, "this is not two word flags")

	echoCmd := &Command{
		Use:     "echo [string to echo]",
		Aliases: []string{"say"},
		Short:   "Echo anything to the screen",
		Long:    "an utterly useless command for testing.",
		Example: "Just run cobra-test echo",
		Run:     emptyRun,
	}

	echoCmd.Flags().String("filename", "", "Enter a filename")
	echoCmd.MarkFlagFilename("filename", "json", "yaml", "yml")
	echoCmd.Flags().String("config", "", "config to use (located in /config/PROFILE/)")
	echoCmd.Flags().SetAnnotation("config", BashCompSubdirsInDir, []string{"config"})

	printCmd := &Command{
		Use:   "print [string to print]",
		Args:  MinimumNArgs(1),
		Short: "Print anything to the screen",
		Long:  "an absolutely utterly useless command for testing.",
		Run:   emptyRun,
	}

	deprecatedCmd := &Command{
		Use:        "deprecated [can't do anything here]",
		Args:       NoArgs,
		Short:      "A command which is deprecated",
		Long:       "an absolutely utterly useless command for testing deprecation!.",
		Deprecated: "Please use echo instead",
		Run:        emptyRun,
	}

	colonCmd := &Command{
		Use: "cmd:colon",
		Run: emptyRun,
	}

	timesCmd := &Command{
		Use:        "times [# times] [string to echo]",
		SuggestFor: []string{"counts"},
		Args:       OnlyValidArgs,
		ValidArgs:  []string{"one", "two", "three", "four"},
		Short:      "Echo anything to the screen more times",
		Long:       "a slightly useless command for testing.",
		Run:        emptyRun,
	}

	echoCmd.AddCommand(timesCmd)
	rootCmd.AddCommand(echoCmd, printCmd, deprecatedCmd, colonCmd)

	buf := new(bytes.Buffer)
	rootCmd.GenBashCompletion(buf)
	output := buf.String()

	// If available, run shellcheck against the script.
	if err := exec.Command("which", "shellcheck").Run(); err != nil {
		return
	}
	if err := runShellCheck(output); err != nil {
		t.Fatalf("shellcheck failed: %v", err)
	}
}
