package pflag

import (
	"io/ioutil"
	"testing"
)

func TestShorthandMultiChars(t *testing.T) {
	f := NewFlagSet("shorthand", ContinueOnError)
	f.AllowMultiCharsShorthand = true
	if f.Parsed() {
		t.Error("f.Parse() = true before Parse")
	}
	boolaFlag := f.BoolP("boola", "a", false, "bool value")
	boolbFlag := f.BoolP("boolb", "b", false, "bool2 value")
	stringaFlag := f.StringP("stringa", "s", "0", "string value")
	stringzFlag := f.StringP("stringz", "z", "0", "string value")
	stringaaFlag := f.StringP("stringaa", "aa", "0", "string value")
	stringzzFlag := f.StringP("stringzz", "zz", "0", "string value")
	extra := "interspersed-argument"
	notaflag := "--i-look-like-a-flag"
	args := []string{
		"-a=true",
		extra,
		"-z=something",
		"-aa",
		"multi-chars-shorthand-aa",
		"-zz=multi-chars-shorthand-zz",
		"--",
		notaflag,
	}
	f.SetOutput(ioutil.Discard)
	if err := f.Parse(args); err != nil {
		t.Error("expected no error, got ", err)
	}
	if !f.Parsed() {
		t.Error("f.Parse() = false after Parse")
	}
	if *boolaFlag != true {
		t.Error("boola flag should be true, is ", *boolaFlag)
	}
	if *boolbFlag != false {
		t.Error("boolb flag should be false, is ", *boolbFlag)
	}
	if *stringaFlag != "0" {
		t.Error("stringa flag should be `0`, is ", *stringaFlag)
	}
	if *stringzFlag != "something" {
		t.Error("stringz flag should be `something`, is ", *stringzFlag)
	}
	if *stringaaFlag != "multi-chars-shorthand-aa" {
		t.Error("stringaa flag should be `multi-chars-shorthand-aa`, is ", *stringaaFlag)
	}
	if *stringzzFlag != "multi-chars-shorthand-zz" {
		t.Error("stringzz flag should be `multi-chars-shorthand-zz`, is ", *stringzzFlag)
	}
	if len(f.Args()) != 2 {
		t.Error("expected one argument, got", len(f.Args()))
	} else if f.Args()[0] != extra {
		t.Errorf("expected argument %q got %q", extra, f.Args()[0])
	} else if f.Args()[1] != notaflag {
		t.Errorf("expected argument %q got %q", notaflag, f.Args()[1])
	}
	if f.ArgsLenAtDash() != 1 {
		t.Errorf("expected argsLenAtDash %d got %d", f.ArgsLenAtDash(), 1)
	}
}

func TestShorthandLookupMultiChars(t *testing.T) {
	f := NewFlagSet("shorthand", ContinueOnError)
	f.AllowMultiCharsShorthand = true
	if f.Parsed() {
		t.Error("f.Parse() = true before Parse")
	}
	f.BoolP("boola", "a", false, "bool value")
	f.BoolP("boolb", "b", false, "bool2 value")
	f.StringP("stringa", "s", "0", "string value")
	f.StringP("stringz", "z", "0", "string value")
	f.StringP("stringaa", "aa", "0", "string value")
	f.StringP("stringzz", "zz", "0", "string value")
	extra := "interspersed-argument"
	notaflag := "--i-look-like-a-flag"
	args := []string{
		"-a=true",
		extra,
		"-z=something",
		"-aa",
		"multi-chars-shorthand-aa",
		"-zz=multi-chars-shorthand-zz",
		"--",
		notaflag,
	}
	f.SetOutput(ioutil.Discard)
	if err := f.Parse(args); err != nil {
		t.Error("expected no error, got ", err)
	}
	if !f.Parsed() {
		t.Error("f.Parse() = false after Parse")
	}
	stringaaFlag := f.ShorthandLookup("aa")
	if stringaaFlag == nil {
		t.Fatal("stringaa flag should not be nil")
	}
	if stringaaFlag.Value.String() != "multi-chars-shorthand-aa" {
		t.Error("stringaa flag should be `multi-chars-shorthand-aa`, is ", stringaaFlag.Value.String())
	}
	stringzzFlag := f.ShorthandLookup("zz")
	if stringzzFlag == nil {
		t.Fatal("stringzz flag should not be nil")
	}
	if stringzzFlag.Value.String() != "multi-chars-shorthand-zz" {
		t.Error("stringzz flag should be `multi-chars-shorthand-zz`, is ", stringzzFlag.Value.String())
	}
	if len(f.Args()) != 2 {
		t.Error("expected one argument, got", len(f.Args()))
	} else if f.Args()[0] != extra {
		t.Errorf("expected argument %q got %q", extra, f.Args()[0])
	} else if f.Args()[1] != notaflag {
		t.Errorf("expected argument %q got %q", notaflag, f.Args()[1])
	}
	if f.ArgsLenAtDash() != 1 {
		t.Errorf("expected argsLenAtDash %d got %d", f.ArgsLenAtDash(), 1)
	}
}
