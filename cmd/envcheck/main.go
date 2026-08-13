// Command envcheck is a fast, cross-platform CLI for validating,
// analyzing, and security-checking .env files, designed for both local
// developer use and CI/CD pipelines.
package main

import (
	"fmt"
	"os"

	"github.com/rizkiromadon/envcheck/internal/app"
	"github.com/rizkiromadon/envcheck/internal/exitcode"
	"github.com/rizkiromadon/envcheck/internal/formatter"
	"github.com/rizkiromadon/envcheck/internal/validator"
)

var version = "dev"

const usageText = `EnvCheck — validate, analyze, and security-check .env files

USAGE:
  envcheck [file] [flags]
  envcheck check [file] [flags]
  envcheck doctor [file] [flags]

  If [file] is omitted, EnvCheck looks for a file named ".env" in the
  current directory.

FLAGS:
  --compare <file>     Compare against an .env.example (or similar) file
  --schema <file>      Validate against a schema file
  --json                Output machine-readable JSON instead of text
  --strict              Treat warnings as failing conditions (non-zero exit)
  --quiet               Only print failures and the summary
  --no-color             Disable ANSI colors in terminal output
  --no-git               Skip the Git tracking check
  --no-security          Skip the security scanner
  --allow-empty          Do not flag empty values
  -h, --help             Show this help message
  -v, --version           Show version information

EXIT CODES:
  0   valid, no errors or critical security findings
  1   validation error (syntax, duplicate keys, schema violations, etc.)
  2   security issue (critical secret detected, or any in --strict mode)
  3   usage error (bad flags, missing/unreadable file)
  4   internal error

EXAMPLES:
  envcheck
  envcheck .env
  envcheck check .env
  envcheck .env --compare .env.example
  envcheck .env --schema .env.schema
  envcheck .env --json --strict
`

type cliOptions struct {
	targetFile string
	compareTo  string
	schemaFile string
	jsonOut    bool
	strict     bool
	quiet      bool
	noColor    bool
	noGit      bool
	noSecurity bool
	allowEmpty bool
	showHelp   bool
	showVer    bool
}

func main() {
	os.Exit(run(os.Args[1:]))
}

// run parses args, executes the EnvCheck pipeline, writes the report, and
// returns the process exit code.
func run(args []string) int {
	opts, err := parseArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "envcheck: "+err.Error())
		fmt.Fprintln(os.Stderr, "\nRun 'envcheck --help' for usage.")
		return exitcode.UsageError
	}

	if opts.showHelp {
		fmt.Print(usageText)
		return exitcode.OK
	}
	if opts.showVer {
		fmt.Printf("EnvCheck v%s\n", version)
		return exitcode.OK
	}

	if opts.targetFile == "" {
		opts.targetFile = ".env"
	}

	if _, err := os.Stat(opts.targetFile); err != nil {
		fmt.Fprintf(os.Stderr, "envcheck: cannot access '%s': %v\n", opts.targetFile, err)
		return exitcode.UsageError
	}

	cfg := app.Config{
		Version:      version,
		TargetFile:   opts.targetFile,
		CompareTo:    opts.compareTo,
		SchemaFile:   opts.schemaFile,
		CheckGit:     !opts.noGit,
		SkipSecurity: opts.noSecurity,
		ValidatorOpts: validator.Options{
			AllowEmptyValues: opts.allowEmpty,
		},
	}

	report, err := app.Run(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "envcheck: %v\n", err)
		return exitcode.InternalError
	}

	if opts.jsonOut {
		if err := formatter.WriteJSON(os.Stdout, report); err != nil {
			fmt.Fprintf(os.Stderr, "envcheck: failed to write JSON: %v\n", err)
			return exitcode.InternalError
		}
	} else {
		formatter.WriteHuman(os.Stdout, report, formatter.HumanOptions{
			NoColor: opts.noColor,
			Quiet:   opts.quiet,
			Version: version,
		})
	}

	return exitcode.Resolve(report, exitcode.Options{Strict: opts.strict})
}

// parseArgs is a small, dependency-free CLI parser supporting an optional
// leading subcommand alias, an optional positional file argument, long
// flags (some with a value), and short aliases -h/-v.
func parseArgs(args []string) (cliOptions, error) {
	var o cliOptions

	if len(args) > 0 && (args[0] == "check" || args[0] == "doctor") {
		args = args[1:]
	}

	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			o.showHelp = true
		case "-v", "--version":
			o.showVer = true
		case "--json":
			o.jsonOut = true
		case "--strict":
			o.strict = true
		case "--quiet":
			o.quiet = true
		case "--no-color":
			o.noColor = true
		case "--no-git":
			o.noGit = true
		case "--no-security":
			o.noSecurity = true
		case "--allow-empty":
			o.allowEmpty = true
		case "--compare":
			v, next, err := requireValue(args, i, "--compare")
			if err != nil {
				return o, err
			}
			o.compareTo = v
			i = next
		case "--schema":
			v, next, err := requireValue(args, i, "--schema")
			if err != nil {
				return o, err
			}
			o.schemaFile = v
			i = next
		default:
			if len(a) > 0 && a[0] == '-' {
				return o, fmt.Errorf("unknown flag: %s", a)
			}
			if o.targetFile != "" {
				return o, fmt.Errorf("unexpected extra argument: %s", a)
			}
			o.targetFile = a
		}
	}

	return o, nil
}

// requireValue returns the value following the flag at index i, or an
// error if no value follows.
func requireValue(args []string, i int, flagName string) (string, int, error) {
	if i+1 >= len(args) {
		return "", i, fmt.Errorf("%s requires a value", flagName)
	}
	return args[i+1], i + 1, nil
}
