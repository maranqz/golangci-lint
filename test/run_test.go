package test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	_ "github.com/valyala/quicktemplate"

	"github.com/golangci/golangci-lint/pkg/exitcodes"
	"github.com/golangci/golangci-lint/test/testshared"
)

const minimalPkg = "minimalpkg"

func TestAutogeneratedNoIssues(t *testing.T) {
	testshared.NewRunnerBuilder(t).
		WithTargetPath(testdataDir, "autogenerated").
		Runner().
		Install().
		Run().
		ExpectNoIssues()
}

func TestEmptyDirRun(t *testing.T) {
	testshared.NewRunnerBuilder(t).
		WithEnviron("GO111MODULE=off").
		WithTargetPath(testdataDir, "nogofiles").
		Runner().
		Install().
		Run().
		ExpectExitCode(exitcodes.NoGoFiles).
		ExpectOutputContains(": no go files to analyze")
}

func TestNotExistingDirRun(t *testing.T) {
	testshared.NewRunnerBuilder(t).
		WithEnviron("GO111MODULE=off").
		WithTargetPath(testdataDir, "no_such_dir").
		Runner().
		Install().
		Run().
		ExpectExitCode(exitcodes.Failure).
		ExpectOutputContains("cannot find package").
		ExpectOutputContains(testshared.NormalizeFileInString("/testdata/no_such_dir"))
}

func TestSymlinkLoop(t *testing.T) {
	testshared.NewRunnerBuilder(t).
		WithTargetPath(testdataDir, "symlink_loop", "...").
		Runner().
		Install().
		Run().
		ExpectNoIssues()
}

// TODO(ldez): remove this in v2.
func TestDeadline(t *testing.T) {
	projectRoot := filepath.Join("..", "...")

	testshared.NewRunnerBuilder(t).
		WithArgs("--deadline=1ms").
		WithTargetPath(projectRoot).
		Runner().
		Install().
		Run().
		ExpectExitCode(exitcodes.Timeout).
		ExpectOutputContains(`Timeout exceeded: try increasing it by passing --timeout option`)
}

func TestTimeout(t *testing.T) {
	projectRoot := filepath.Join("..", "...")

	testshared.NewRunnerBuilder(t).
		WithArgs("--timeout=1ms").
		WithTargetPath(projectRoot).
		Runner().
		Install().
		Run().
		ExpectExitCode(exitcodes.Timeout).
		ExpectOutputContains(`Timeout exceeded: try increasing it by passing --timeout option`)
}

func TestTimeoutInConfig(t *testing.T) {
	cases := []struct {
		cfg string
	}{
		{
			cfg: `
				run:
					deadline: 1ms
			`,
		},
		{
			cfg: `
				run:
					timeout: 1ms
			`,
		},
		{
			// timeout should override deadline
			cfg: `
				run:
					deadline: 100s
					timeout: 1ms
			`,
		},
	}

	testshared.InstallGolangciLint(t)

	for _, c := range cases {
		// Run with disallowed option set only in config
		testshared.NewRunnerBuilder(t).
			WithConfig(c.cfg).
			WithTargetPath(testdataDir, minimalPkg).
			Runner().
			Run().
			ExpectExitCode(exitcodes.Timeout).
			ExpectOutputContains(`Timeout exceeded: try increasing it by passing --timeout option`)
	}
}

func TestTestsAreLintedByDefault(t *testing.T) {
	testshared.NewRunnerBuilder(t).
		WithTargetPath(testdataDir, "withtests").
		Runner().
		Install().
		Run().
		ExpectHasIssue("don't use `init` function")
}

func TestCgoOk(t *testing.T) {
	testshared.NewRunnerBuilder(t).
		WithNoConfig().
		WithArgs(
			"--timeout=3m",
			"--enable-all",
			"-D",
			"nosnakecase,gci,gofactory",
		).
		WithTargetPath(testdataDir, "cgo").
		Runner().
		Install().
		Run().
		ExpectNoIssues()
}

func TestCgoWithIssues(t *testing.T) {
	testshared.InstallGolangciLint(t)

	testCases := []struct {
		desc     string
		args     []string
		dir      string
		expected string
	}{
		{
			desc:     "govet",
			args:     []string{"--no-config", "--disable-all", "-Egovet"},
			dir:      "cgo_with_issues",
			expected: "Printf format %t has arg cs of wrong type",
		},
		{
			desc:     "staticcheck",
			args:     []string{"--no-config", "--disable-all", "-Estaticcheck"},
			dir:      "cgo_with_issues",
			expected: "SA5009: Printf format %t has arg #1 of wrong type",
		},
		{
			desc:     "gofmt",
			args:     []string{"--no-config", "--disable-all", "-Egofmt"},
			dir:      "cgo_with_issues",
			expected: "File is not `gofmt`-ed with `-s` (gofmt)",
		},
		{
			desc:     "revive",
			args:     []string{"--no-config", "--disable-all", "-Erevive"},
			dir:      "cgo_with_issues",
			expected: "indent-error-flow: if block ends with a return statement",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			testshared.NewRunnerBuilder(t).
				WithArgs(test.args...).
				WithTargetPath(testdataDir, test.dir).
				Runner().
				Run().
				ExpectHasIssue(test.expected)
		})
	}
}

// https://pkg.go.dev/cmd/compile#hdr-Compiler_Directives
func TestLineDirective(t *testing.T) {
	testshared.InstallGolangciLint(t)

	testCases := []struct {
		desc       string
		args       []string
		configPath string
		targetPath string
		expected   string
	}{
		{
			desc: "dupl",
			args: []string{
				"-Edupl",
				"--disable-all",
			},
			configPath: "testdata/linedirective/dupl.yml",
			targetPath: "linedirective",
			expected:   "21-23 lines are duplicate of `testdata/linedirective/hello.go:25-27` (dupl)",
		},
		{
			desc: "gofmt",
			args: []string{
				"-Egofmt",
				"--disable-all",
			},
			targetPath: "linedirective",
			expected:   "File is not `gofmt`-ed with `-s` (gofmt)",
		},
		{
			desc: "goimports",
			args: []string{
				"-Egoimports",
				"--disable-all",
			},
			targetPath: "linedirective",
			expected:   "File is not `goimports`-ed (goimports)",
		},
		{
			desc: "gomodguard",
			args: []string{
				"-Egomodguard",
				"--disable-all",
			},
			configPath: "testdata/linedirective/gomodguard.yml",
			targetPath: "linedirective",
			expected: "import of package `golang.org/x/tools/go/analysis` is blocked because the module is not " +
				"in the allowed modules list. (gomodguard)",
		},
		{
			desc: "lll",
			args: []string{
				"-Elll",
				"--disable-all",
			},
			configPath: "testdata/linedirective/lll.yml",
			targetPath: "linedirective",
			expected:   "line is 57 characters (lll)",
		},
		{
			desc: "misspell",
			args: []string{
				"-Emisspell",
				"--disable-all",
			},
			configPath: "",
			targetPath: "linedirective",
			expected:   "is a misspelling of `language` (misspell)",
		},
		{
			desc: "wsl",
			args: []string{
				"-Ewsl",
				"--disable-all",
			},
			configPath: "",
			targetPath: "linedirective",
			expected:   "block should not start with a whitespace (wsl)",
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			testshared.NewRunnerBuilder(t).
				WithArgs(test.args...).
				WithTargetPath(testdataDir, test.targetPath).
				WithConfigFile(test.configPath).
				Runner().
				Run().
				ExpectHasIssue(test.expected)
		})
	}
}

// https://pkg.go.dev/cmd/compile#hdr-Compiler_Directives
func TestLineDirectiveProcessedFiles(t *testing.T) {
	testCases := []struct {
		desc     string
		args     []string
		target   string
		expected []string
	}{
		{
			desc: "lite loading",
			args: []string{
				"--print-issued-lines=false",
				"--exclude-use-default=false",
				"-Erevive",
			},
			target: "quicktemplate",
			expected: []string{
				"testdata/quicktemplate/hello.qtpl.go:10:1: package-comments: should have a package comment (revive)",
				"testdata/quicktemplate/hello.qtpl.go:26:1: exported: exported function StreamHello should have comment or be unexported (revive)",
				"testdata/quicktemplate/hello.qtpl.go:39:1: exported: exported function WriteHello should have comment or be unexported (revive)",
				"testdata/quicktemplate/hello.qtpl.go:50:1: exported: exported function Hello should have comment or be unexported (revive)",
			},
		},
		{
			desc: "full loading",
			args: []string{
				"--print-issued-lines=false",
				"--exclude-use-default=false",
				"-Erevive,govet",
			},
			target: "quicktemplate",
			expected: []string{
				"testdata/quicktemplate/hello.qtpl.go:10:1: package-comments: should have a package comment (revive)",
				"testdata/quicktemplate/hello.qtpl.go:26:1: exported: exported function StreamHello should have comment or be unexported (revive)",
				"testdata/quicktemplate/hello.qtpl.go:39:1: exported: exported function WriteHello should have comment or be unexported (revive)",
				"testdata/quicktemplate/hello.qtpl.go:50:1: exported: exported function Hello should have comment or be unexported (revive)",
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			testshared.NewRunnerBuilder(t).
				WithNoConfig().
				WithArgs(test.args...).
				WithTargetPath(testdataDir, test.target).
				Runner().
				Install().
				Run().
				ExpectExitCode(exitcodes.IssuesFound).
				ExpectOutputContains(test.expected...)
		})
	}
}

func TestUnsafeOk(t *testing.T) {
	testshared.NewRunnerBuilder(t).
		WithNoConfig().
		WithArgs(
			"--enable-all",
			"-D",
			"gofactory",
		).
		WithTargetPath(testdataDir, "unsafe").
		Runner().
		Install().
		Run().
		ExpectNoIssues()
}

func TestSortedResults(t *testing.T) {
	testCases := []struct {
		opt  string
		want string
	}{
		{
			opt: "--sort-results=false",
			want: "testdata/sort_results/main.go:15:13: Error return value is not checked (errcheck)" + "\n" +
				"testdata/sort_results/main.go:12:5: var `db` is unused (unused)",
		},
		{
			opt: "--sort-results=true",
			want: "testdata/sort_results/main.go:12:5: var `db` is unused (unused)" + "\n" +
				"testdata/sort_results/main.go:15:13: Error return value is not checked (errcheck)",
		},
	}

	testshared.InstallGolangciLint(t)

	for _, test := range testCases {
		test := test
		t.Run(test.opt, func(t *testing.T) {
			t.Parallel()

			testshared.NewRunnerBuilder(t).
				WithNoConfig().
				WithArgs("--print-issued-lines=false", test.opt).
				WithTargetPath(testdataDir, "sort_results").
				Runner().
				Run().
				ExpectExitCode(exitcodes.IssuesFound).ExpectOutputEq(test.want + "\n")
		})
	}
}

func TestSkippedDirsNoMatchArg(t *testing.T) {
	dir := filepath.Join(testdataDir, "skipdirs", "skip_me", "nested")

	testshared.NewRunnerBuilder(t).
		WithNoConfig().
		WithArgs(
			"--print-issued-lines=false",
			"--skip-dirs", dir,
			"-Erevive",
		).
		WithTargetPath(dir).
		Runner().
		Install().
		Run().
		ExpectExitCode(exitcodes.IssuesFound).
		ExpectOutputEq("testdata/skipdirs/skip_me/nested/with_issue.go:8:9: " +
			"indent-error-flow: if block ends with a return statement, so drop this else and outdent its block (revive)\n")
}

func TestSkippedDirsTestdata(t *testing.T) {
	testshared.NewRunnerBuilder(t).
		WithNoConfig().
		WithArgs(
			"--print-issued-lines=false",
			"-Erevive",
		).
		WithTargetPath(testdataDir, "skipdirs", "...").
		Runner().
		Install().
		Run().
		ExpectNoIssues() // all was skipped because in testdata
}

func TestIdentifierUsedOnlyInTests(t *testing.T) {
	testshared.NewRunnerBuilder(t).
		WithNoConfig().
		WithArgs("--disable-all", "-Eunused").
		WithTargetPath(testdataDir, "used_only_in_tests").
		Runner().
		Install().
		Run().
		ExpectNoIssues()
}

func TestUnusedCheckExported(t *testing.T) {
	testshared.NewRunnerBuilder(t).
		WithConfigFile("testdata_etc/unused_exported/golangci.yml").
		WithTargetPath("testdata_etc/unused_exported/...").
		Runner().
		Install().
		Run().
		ExpectNoIssues()
}

func TestConfigFileIsDetected(t *testing.T) {
	testshared.InstallGolangciLint(t)

	testCases := []struct {
		desc       string
		targetPath string
	}{
		{
			desc:       "explicit",
			targetPath: filepath.Join(testdataDir, "withconfig", "pkg"),
		},
		{
			desc:       "recursive",
			targetPath: filepath.Join(testdataDir, "withconfig", "..."),
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			testshared.NewRunnerBuilder(t).
				// WithNoConfig().
				WithTargetPath(test.targetPath).
				Runner().
				Run().
				ExpectExitCode(exitcodes.Success).
				// test config contains InternalTest: true, it triggers such output
				ExpectOutputEq("test\n")
		})
	}
}

func TestEnableAllFastAndEnableCanCoexist(t *testing.T) {
	testshared.InstallGolangciLint(t)

	testCases := []struct {
		desc     string
		args     []string
		expected []int
	}{
		{
			desc:     "fast",
			args:     []string{"--fast", "--enable-all", "--enable=typecheck"},
			expected: []int{exitcodes.Success, exitcodes.IssuesFound},
		},
		{
			desc:     "all",
			args:     []string{"--enable-all", "--enable=typecheck"},
			expected: []int{exitcodes.Failure},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			testshared.NewRunnerBuilder(t).
				WithNoConfig().
				WithArgs(test.args...).
				WithTargetPath(testdataDir, minimalPkg).
				Runner().
				Run().
				ExpectExitCode(test.expected...)
		})
	}
}

func TestEnabledPresetsAreNotDuplicated(t *testing.T) {
	testshared.NewRunnerBuilder(t).
		WithNoConfig().
		WithArgs("-v", "-p", "style,bugs").
		WithTargetPath(testdataDir, minimalPkg).
		Runner().
		Install().
		Run().
		ExpectOutputContains("Active presets: [bugs style]")
}

func TestAbsPathDirAnalysis(t *testing.T) {
	dir := filepath.Join("testdata_etc", "abspath") // abs paths don't work with testdata dir
	absDir, err := filepath.Abs(dir)
	require.NoError(t, err)

	testshared.NewRunnerBuilder(t).
		WithNoConfig().
		WithArgs(
			"--print-issued-lines=false",
			"-Erevive",
		).
		WithTargetPath(absDir).
		Runner().
		Install().
		Run().
		ExpectHasIssue("testdata_etc/abspath/with_issue.go:8:9: " +
			"indent-error-flow: if block ends with a return statement, so drop this else and outdent its block (revive)")
}

func TestAbsPathFileAnalysis(t *testing.T) {
	dir := filepath.Join("testdata_etc", "abspath", "with_issue.go") // abs paths don't work with testdata dir
	absDir, err := filepath.Abs(dir)
	require.NoError(t, err)

	testshared.NewRunnerBuilder(t).
		WithNoConfig().
		WithArgs(
			"--print-issued-lines=false",
			"-Erevive",
		).
		WithTargetPath(absDir).
		Runner().
		Install().
		Run().
		ExpectHasIssue("indent-error-flow: if block ends with a return statement, so drop this else and outdent its block (revive)")
}

func TestDisallowedOptionsInConfig(t *testing.T) {
	cases := []struct {
		cfg    string
		option string
	}{
		{
			cfg: `
				ruN:
					Args:
						- 1
			`,
		},
		{
			cfg: `
				run:
					CPUProfilePath: path
			`,
			option: "--cpu-profile-path=path",
		},
		{
			cfg: `
				run:
					MemProfilePath: path
			`,
			option: "--mem-profile-path=path",
		},
		{
			cfg: `
				run:
					TracePath: path
			`,
			option: "--trace-path=path",
		},
		{
			cfg: `
				run:
					Verbose: true
			`,
			option: "-v",
		},
	}

	testshared.InstallGolangciLint(t)

	for _, c := range cases {
		// Run with disallowed option set only in config
		testshared.NewRunnerBuilder(t).
			WithConfig(c.cfg).
			WithTargetPath(testdataDir, minimalPkg).
			Runner().
			Run().
			ExpectExitCode(exitcodes.Failure)

		if c.option == "" {
			continue
		}

		args := []string{c.option, "--fast"}

		// Run with disallowed option set only in command-line
		testshared.NewRunnerBuilder(t).
			WithNoConfig().
			WithArgs(args...).
			WithTargetPath(testdataDir, minimalPkg).
			Runner().
			Run().
			ExpectExitCode(exitcodes.Success)

		// Run with disallowed option set both in command-line and in config

		testshared.NewRunnerBuilder(t).
			WithConfig(c.cfg).
			WithArgs(args...).
			WithTargetPath(testdataDir, minimalPkg).
			Runner().
			Run().
			ExpectExitCode(exitcodes.Failure)
	}
}

func TestPathPrefix(t *testing.T) {
	testCases := []struct {
		desc    string
		args    []string
		pattern string
	}{
		{
			desc:    "empty",
			pattern: "^testdata/withtests/",
		},
		{
			desc:    "prefixed",
			args:    []string{"--path-prefix=cool"},
			pattern: "^cool/testdata/withtests",
		},
	}

	testshared.InstallGolangciLint(t)

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			testshared.NewRunnerBuilder(t).
				WithArgs(test.args...).
				WithTargetPath(testdataDir, "withtests").
				Runner().
				Run().
				ExpectOutputRegexp(test.pattern)
		})
	}
}
