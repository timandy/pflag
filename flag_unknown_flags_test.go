package pflag

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnknownFlagsEmpty(t *testing.T) {
	f := NewFlagSet("test", ContinueOnError)
	f.ParseErrorsAllowlist.UnknownFlags = true
	f.Bool("known", false, "a known flag")
	f.SetOutput(ioutil.Discard)

	require.NoError(t, f.Parse([]string{"--known"}))
	assert.Empty(t, f.UnknownFlags())
}

func TestUnknownFlagsLong(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantUnknown []string
		wantArgs    []string
	}{
		{
			name:        "long with equals",
			args:        []string{"--foo=bar"},
			wantUnknown: []string{"--foo=bar"},
		},
		{
			name:        "long with value",
			args:        []string{"--foo", "bar"},
			wantUnknown: []string{"--foo", "bar"},
		},
		{
			name:        "long followed by flag",
			args:        []string{"--foo", "--known"},
			wantUnknown: []string{"--foo"},
		},
		{
			name:        "long at end",
			args:        []string{"--foo"},
			wantUnknown: []string{"--foo"},
		},
		{
			name:        "long with equals and positional",
			args:        []string{"--foo=bar", "pos"},
			wantUnknown: []string{"--foo=bar", "pos"},
			wantArgs:    []string{"pos"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFlagSet("test", ContinueOnError)
			f.ParseErrorsAllowlist.UnknownFlags = true
			f.Bool("known", false, "a known flag")
			f.SetOutput(ioutil.Discard)

			require.NoError(t, f.Parse(tt.args))
			assert.Equal(t, tt.wantUnknown, f.UnknownFlags())
			if tt.wantArgs != nil {
				assert.Equal(t, tt.wantArgs, f.Args())
			}
		})
	}
}

func TestUnknownFlagsShort(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantUnknown []string
		wantArgs    []string
	}{
		{
			name:        "short with equals",
			args:        []string{"-f=arg"},
			wantUnknown: []string{"-f=arg"},
		},
		{
			name:        "short at end",
			args:        []string{"-f"},
			wantUnknown: []string{"-f"},
		},
		{
			name:        "short with value",
			args:        []string{"-f", "bar"},
			wantUnknown: []string{"-f", "bar"},
		},
		{
			name:        "short followed by flag",
			args:        []string{"-f", "--known"},
			wantUnknown: []string{"-f"},
		},
		{
			name:        "partial shorthand group",
			args:        []string{"-af"},
			wantUnknown: []string{"-f"},
		},
		{
			name:        "multiple unknown in group",
			args:        []string{"-fag"},
			wantUnknown: []string{"-fg"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFlagSet("test", ContinueOnError)
			f.ParseErrorsAllowlist.UnknownFlags = true
			f.BoolP("known", "a", false, "a known flag")
			f.SetOutput(ioutil.Discard)

			require.NoError(t, f.Parse(tt.args))
			assert.Equal(t, tt.wantUnknown, f.UnknownFlags())
			if tt.wantArgs != nil {
				assert.Equal(t, tt.wantArgs, f.Args())
			}
		})
	}
}

func TestUnknownFlagsSpecialCases(t *testing.T) {
	t.Run("help long not collected (consumed)", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.SetOutput(ioutil.Discard)
		f.Usage = func() {}

		assert.ErrorIs(t, f.Parse([]string{"--help"}), ErrHelp)
		assert.Empty(t, f.UnknownFlags())
	})

	t.Run("help short not collected (consumed)", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.SetOutput(ioutil.Discard)
		f.Usage = func() {}

		assert.ErrorIs(t, f.Parse([]string{"-h"}), ErrHelp)
		assert.Empty(t, f.UnknownFlags())
	})

	t.Run("double dash collected", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.SetOutput(ioutil.Discard)

		require.NoError(t, f.Parse([]string{"--"}))
		assert.Equal(t, []string{"--"}, f.UnknownFlags())
	})

	t.Run("double dash with post-terminator args collected", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.SetOutput(ioutil.Discard)

		require.NoError(t, f.Parse([]string{"--", "--foo", "pos"}))
		assert.Equal(t, []string{"--", "--foo", "pos"}, f.UnknownFlags())
		assert.Equal(t, []string{"--foo", "pos"}, f.Args())
	})
}

func TestUnknownFlagsEdgeCases(t *testing.T) {
	t.Run("allowlist false returns error, no collection", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.SetOutput(ioutil.Discard)

		assert.Error(t, f.Parse([]string{"--foo"}))
		assert.Empty(t, f.UnknownFlags())
	})

	t.Run("whitelist backwards compat", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsWhitelist.UnknownFlags = true
		f.SetOutput(ioutil.Discard)

		require.NoError(t, f.Parse([]string{"--foo=bar"}))
		assert.Equal(t, []string{"--foo=bar"}, f.UnknownFlags())
	})

	t.Run("mixed known and unknown", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.BoolP("verbose", "v", false, "verbose")
		f.String("output", "", "output file")
		f.SetOutput(ioutil.Discard)

		args := []string{"--verbose", "--unknown1=val", "-v", "-x", "--output", "file.txt", "--unknown2", "pos"}
		require.NoError(t, f.Parse(args))
		assert.Equal(t, []string{"--unknown1=val", "-x", "--unknown2", "pos"}, f.UnknownFlags())
		assert.Empty(t, f.Args())
	})

	t.Run("parse resets unknownFlags", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.SetOutput(ioutil.Discard)

		require.NoError(t, f.Parse([]string{"--foo"}))
		assert.Len(t, f.UnknownFlags(), 1)

		require.NoError(t, f.Parse([]string{}))
		assert.Empty(t, f.UnknownFlags())
	})

	t.Run("top-level UnknownFlags function", func(t *testing.T) {
		ResetForTesting(func() {})
		f := GetCommandLine()
		f.ParseErrorsAllowlist.UnknownFlags = true

		require.NoError(t, f.Parse([]string{"--foo"}))
		assert.Equal(t, []string{"--foo"}, UnknownFlags())
	})

	t.Run("positional arg collected", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.SetOutput(ioutil.Discard)

		require.NoError(t, f.Parse([]string{"arg"}))
		assert.Equal(t, []string{"arg"}, f.UnknownFlags())
		assert.Equal(t, []string{"arg"}, f.Args())
	})

	t.Run("go test flags collected", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.SetOutput(ioutil.Discard)

		require.NoError(t, f.Parse([]string{"-test.v"}))
		assert.Equal(t, []string{"-test.v"}, f.UnknownFlags())
	})

	t.Run("interspersed false: collects remaining args", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.SetInterspersed(false)
		f.Bool("known", false, "a known flag")
		f.SetOutput(ioutil.Discard)

		require.NoError(t, f.Parse([]string{"--known", "arg", "--unknown"}))
		assert.Equal(t, []string{"arg", "--unknown"}, f.UnknownFlags())
		assert.Equal(t, []string{"arg", "--unknown"}, f.Args())
	})

	t.Run("interspersed true: positional between flags", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.Bool("known", false, "a known flag")
		f.SetOutput(ioutil.Discard)

		require.NoError(t, f.Parse([]string{"--known", "pos1", "--unknown", "pos2"}))
		assert.Equal(t, []string{"pos1", "--unknown", "pos2"}, f.UnknownFlags())
		assert.Equal(t, []string{"pos1"}, f.Args())
	})

	t.Run("interspersed true: multiple positionals", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.Bool("known", false, "a known flag")
		f.SetOutput(ioutil.Discard)

		require.NoError(t, f.Parse([]string{"a", "b", "c"}))
		assert.Equal(t, []string{"a", "b", "c"}, f.UnknownFlags())
		assert.Equal(t, []string{"a", "b", "c"}, f.Args())
	})

	t.Run("interspersed true: positional then unknown flag then positional", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.String("output", "", "output file")
		f.SetOutput(ioutil.Discard)

		require.NoError(t, f.Parse([]string{"pos1", "--foo=bar", "--output", "out.txt", "pos2"}))
		assert.Equal(t, []string{"pos1", "--foo=bar", "pos2"}, f.UnknownFlags())
		assert.Equal(t, []string{"pos1", "pos2"}, f.Args())
	})

	t.Run("ParseAll collects unknown flags", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.Bool("known", false, "a known flag")
		f.SetOutput(ioutil.Discard)

		err := f.ParseAll([]string{"--unknown=val", "--known"}, func(_ *Flag, _ string) error {
			return nil
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"--unknown=val"}, f.UnknownFlags())
	})

	t.Run("NormalizeFunc known after normalize not collected", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.SetNormalizeFunc(func(_ *FlagSet, name string) NormalizedName {
			return NormalizedName(strings.ReplaceAll(name, "_", "-"))
		})
		f.Bool("my-flag", false, "a flag with dash")
		f.SetOutput(ioutil.Discard)

		require.NoError(t, f.Parse([]string{"--my_flag"}))
		assert.Empty(t, f.UnknownFlags())
	})
}

func TestUnknownFlagsCombinedShortGroup(t *testing.T) {
	t.Run("known bool in combined group", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.BoolP("alpha", "a", false, "a bool flag")
		f.SetOutput(ioutil.Discard)

		require.NoError(t, f.Parse([]string{"-xaf", "value", "-x", "v1", "-a", "v2", "-f", "c3"}))
		assert.Equal(t, []string{"-xf", "value", "-x", "v1", "v2", "-f", "c3"}, f.UnknownFlags())
		alpha, err := f.GetBool("alpha")
		require.NoError(t, err)
		assert.True(t, alpha)
	})

	t.Run("known non-bool in combined group", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.StringP("alpha", "a", "", "a string flag")
		f.SetOutput(ioutil.Discard)

		require.NoError(t, f.Parse([]string{"-xf", "value", "-x", "v1", "-a", "v2", "-f", "c3"}))
		assert.Equal(t, []string{"-xf", "value", "-x", "v1", "-f", "c3"}, f.UnknownFlags())
		alpha, err := f.GetString("alpha")
		require.NoError(t, err)
		assert.Equal(t, "v2", alpha)
	})

	t.Run("known non-bool consumes remaining shorthands as value", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.StringP("alpha", "a", "", "a string flag")
		f.SetOutput(ioutil.Discard)

		require.NoError(t, f.Parse([]string{"-xaf", "value", "-xd", "v1", "-f", "c3"}))
		assert.Equal(t, []string{"-x", "value", "-xd", "v1", "-f", "c3"}, f.UnknownFlags())
		alpha, err := f.GetString("alpha")
		require.NoError(t, err)
		assert.Equal(t, "f", alpha)
	})

	t.Run("known non-bool takes flag-like arg as value", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.StringP("alpha", "a", "", "a string flag")
		f.SetOutput(ioutil.Discard)

		require.NoError(t, f.Parse([]string{"-xf", "value", "-a", "-xd", "v1", "-f", "c3"}))
		assert.Equal(t, []string{"-xf", "value", "v1", "-f", "c3"}, f.UnknownFlags())
		alpha, err := f.GetString("alpha")
		require.NoError(t, err)
		assert.Equal(t, "-xd", alpha)
	})

	t.Run("known non-bool takes next positional as value", func(t *testing.T) {
		f := NewFlagSet("test", ContinueOnError)
		f.ParseErrorsAllowlist.UnknownFlags = true
		f.StringP("alpha", "a", "", "a string flag")
		f.SetOutput(ioutil.Discard)

		require.NoError(t, f.Parse([]string{"-xf", "value", "-a", "xxx", "-xd", "v1", "-f", "c3"}))
		assert.Equal(t, []string{"-xf", "value", "-xd", "v1", "-f", "c3"}, f.UnknownFlags())
		alpha, err := f.GetString("alpha")
		require.NoError(t, err)
		assert.Equal(t, "xxx", alpha)
	})
}
