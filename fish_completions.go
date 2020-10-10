package cobra

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

func genFishComp(buf *bytes.Buffer, name string, includeDesc bool) {
	// Variables should not contain a '-' or ':' character
	nameForVar := name
	nameForVar = strings.Replace(nameForVar, "-", "_", -1)
	nameForVar = strings.Replace(nameForVar, ":", "_", -1)

	compCmd := ShellCompRequestCmd
	if !includeDesc {
		compCmd = ShellCompNoDescRequestCmd
	}
	buf.WriteString(fmt.Sprintf("# fish completion for %-36s -*- shell-script -*-\n", name))
	buf.WriteString(fmt.Sprintf(`
function __%[1]s_debug
    set file "$BASH_COMP_DEBUG_FILE"
    if test -n "$file"
        echo "$argv" >> $file
    end
end

function __%[1]s_perform_completion
    __%[1]s_debug "Starting __%[1]s_perform_completion"

    set args (string split -- " " (commandline -c))
    set lastArg "$args[-1]"

    __%[1]s_debug "args: $args"
    __%[1]s_debug "last arg: $lastArg"

    set emptyArg ""
    if test -z "$lastArg"
        __%[1]s_debug "Setting emptyArg"
        set emptyArg \"\"
    end
    __%[1]s_debug "emptyArg: $emptyArg"

    if not type -q "$args[1]"
        # This can happen when "complete --do-complete %[2]s" is called when running this script.
        __%[1]s_debug "Cannot find $args[1]. No completions."
        return
    end

    set requestComp "$args[1] %[3]s $args[2..-1] $emptyArg"
    __%[1]s_debug "Calling $requestComp"

    set results (eval $requestComp 2> /dev/null)
    set comps $results[1..-2]
    set directiveLine $results[-1]

    # For Fish, when completing a flag with an = (e.g., <program> -n=<TAB>)
    # completions must be prefixed with the flag
    set flagPrefix (string match -r -- '-.*=' "$lastArg")

    __%[1]s_debug "Comps: $comps"
    __%[1]s_debug "DirectiveLine: $directiveLine"
    __%[1]s_debug "flagPrefix: $flagPrefix"

    for comp in $comps
        printf "%%s%%s\n" "$flagPrefix" "$comp"
    end

    printf "%%s\n" "$directiveLine"
end

# This function does two things:
# - Obtain the completions and store them in the global __%[1]s_comp_results
# - Return false if file completion should be performed
function __%[1]s_prepare_completions
    __%[1]s_debug ""
    __%[1]s_debug "========= starting completion logic =========="

    # Start fresh
    set --erase __%[1]s_comp_results

    set results (__%[1]s_perform_completion)
    __%[1]s_debug "Completion results: $results"

    if test -z "$results"
        __%[1]s_debug "No completion, probably due to a failure"
        # Might as well do file completion, in case it helps
        return 1
    end

    set directive (string sub --start 2 $results[-1])
    set --global __%[1]s_comp_results $results[1..-2]

    __%[1]s_debug "Completions are: $__%[1]s_comp_results"
    __%[1]s_debug "Directive is: $directive"

    set shellCompDirectiveError %[4]d
    set shellCompDirectiveNoSpace %[5]d
    set shellCompDirectiveNoFileComp %[6]d
    set shellCompDirectiveFilterFileExt %[7]d
    set shellCompDirectiveFilterDirs %[8]d

    if test -z "$directive"
        set directive 0
    end

    set compErr (math (math --scale 0 $directive / $shellCompDirectiveError) %% 2)
    if test $compErr -eq 1
        __%[1]s_debug "Received error directive: aborting."
        # Might as well do file completion, in case it helps
        return 1
    end

    set filefilter (math (math --scale 0 $directive / $shellCompDirectiveFilterFileExt) %% 2)
    set dirfilter (math (math --scale 0 $directive / $shellCompDirectiveFilterDirs) %% 2)
    if test $filefilter -eq 1; or test $dirfilter -eq 1
        __%[1]s_debug "File extension filtering or directory filtering not supported"
        # Do full file completion instead
        return 1
    end

    set nospace (math (math --scale 0 $directive / $shellCompDirectiveNoSpace) %% 2)
    set nofiles (math (math --scale 0 $directive / $shellCompDirectiveNoFileComp) %% 2)

    __%[1]s_debug "nospace: $nospace, nofiles: $nofiles"

    # If we want to prevent a space, or if file completion is NOT disabled,
    # we need to count the number of valid completions.
    # To do so, we will filter on prefix as the completions we have received
    # may not already be filtered so as to allow fish to match on different
    # criteria than prefix.
    if test $nospace -ne 0; or test $nofiles -eq 0
        set prefix (commandline -t)
        __%[1]s_debug "prefix: $prefix"

        set completions
        for comp in $__%[1]s_comp_results
            if test (string match -e -r "^$prefix" "$comp")
                set -a completions $comp
            end
        end
        set --global __%[1]s_comp_results $completions
        __%[1]s_debug "Filtered completions are: $__%[1]s_comp_results"

        # Important not to quote the variable for count to work
        set numComps (count $__%[1]s_comp_results)
        __%[1]s_debug "numComps: $numComps"

        if test $numComps -eq 1; and test $nospace -ne 0
            # To support the "nospace" directive we trick the shell
            # by outputting an extra, longer completion.
            # We must first split on \t to get rid of the descriptions because
            # the extra character we add to the fake second completion must be
            # before the description.  We don't need descriptions anyway since
            # there is only a single real completion which the shell will expand
            # immediately.
            __%[1]s_debug "Adding second completion to perform nospace directive"
            set split (string split --max 1 \t $__%[1]s_comp_results[1])
            set --global __%[1]s_comp_results $split[1] $split[1].
            __%[1]s_debug "Completions are now: $__%[1]s_comp_results"
        end

        if test $numComps -eq 0; and test $nofiles -eq 0
            # To be consistent with bash and zsh, we only trigger file
            # completion when there are no other completions
            __%[1]s_debug "Requesting file completion"
            return 1
        end
    end

    return 0
end

# Since Fish completions are only loaded once the user triggers them, we trigger them ourselves
# so we can properly delete any completions provided by another script.
# The space after the the program name is essential to trigger completion for the program
# and not completion of the program name itself.
complete --do-complete "%[2]s " > /dev/null 2>&1
# Using '> /dev/null 2>&1' since '&>' is not supported in older versions of fish.

# Remove any pre-existing completions for the program since we will be handling all of them.
complete -c %[2]s -e

# The call to __%[1]s_prepare_completions will setup __%[1]s_comp_results
# which provides the program's completion choices.
complete -c %[2]s -n '__%[1]s_prepare_completions' -f -a '$__%[1]s_comp_results'

`, nameForVar, name, compCmd,
		ShellCompDirectiveError, ShellCompDirectiveNoSpace, ShellCompDirectiveNoFileComp,
		ShellCompDirectiveFilterFileExt, ShellCompDirectiveFilterDirs))
}

// GenFishCompletion generates fish completion file and writes to the passed writer.
func (c *Command) GenFishCompletion(w io.Writer, includeDesc bool) error {
	buf := new(bytes.Buffer)
	genFishComp(buf, c.Name(), includeDesc)
	_, err := buf.WriteTo(w)
	return err
}

// GenFishCompletionFile generates fish completion file.
func (c *Command) GenFishCompletionFile(filename string, includeDesc bool) error {
	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return c.GenFishCompletion(outFile, includeDesc)
}
