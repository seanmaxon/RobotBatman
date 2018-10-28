package main

import "testing"

var sqnTests = []struct {
	n1      sqn //num, limit, window
	n2      sqn
	action  string
	res     sqn
	truth   bool
	comment string
}{
	{
		n1:      sqn{9, 10, 3},
		n2:      sqn{2, 10, 3},
		action:  "add",
		res:     sqn{1, 10, 3},
		truth:   false,
		comment: "rollover test",
	},
	{
		n1:      sqn{1, 2048, 64},
		n2:      sqn{100, 128, 32},
		action:  "add",
		res:     sqn{101, 2048, 64},
		truth:   false,
		comment: "limit and window of res are taken from n1, not n2",
	},
	{
		n1:      sqn{0, 2048, 64},
		n2:      sqn{1, 2048, 64},
		action:  "subtract",
		res:     sqn{2047, 2048, 64},
		truth:   false,
		comment: "rollover test",
	},
	{
		n1:      sqn{99, 2048, 64},
		n2:      sqn{60, 64, 32},
		action:  "subtract",
		res:     sqn{39, 2048, 64},
		truth:   false,
		comment: "limit and window of res are taken from n1, not n2",
	},
	{
		n1:      sqn{212, 2048, 64},
		n2:      sqn{212, 2048, 64},
		action:  "equalTo",
		res:     sqn{},
		truth:   true,
		comment: "equal",
	},
	{
		n1:      sqn{100, 2048, 64},
		n2:      sqn{90, 2048, 64},
		action:  "equalTo",
		res:     sqn{},
		truth:   false,
		comment: "greater",
	},
	{
		n1:      sqn{50, 2048, 64},
		n2:      sqn{60, 2048, 64},
		action:  "equalTo",
		res:     sqn{},
		truth:   false,
		comment: "lessthan",
	},
	{
		n1:      sqn{1000, 2048, 64},
		n2:      sqn{2000, 2048, 64},
		action:  "equalTo",
		res:     sqn{},
		truth:   false,
		comment: "lessthan beyond window",
	},
	{
		n1:      sqn{212, 2048, 64},
		n2:      sqn{212, 2048, 64},
		action:  "greaterThan",
		res:     sqn{},
		truth:   false,
		comment: "equal",
	},
	{
		n1:      sqn{212, 2048, 64},
		n2:      sqn{213, 2048, 64},
		action:  "greaterThan",
		res:     sqn{},
		truth:   false,
		comment: "one less than",
	},
	{
		n1:      sqn{212, 2048, 64},
		n2:      sqn{211, 2048, 64},
		action:  "greaterThan",
		res:     sqn{},
		truth:   true,
		comment: "one more than",
	},
	{
		n1:      sqn{100, 2048, 64},
		n2:      sqn{90, 2048, 64},
		action:  "greaterThan",
		res:     sqn{},
		truth:   true,
		comment: "greater and within window size",
	},
	{
		n1:      sqn{1000, 2048, 64},
		n2:      sqn{90, 2048, 64},
		action:  "greaterThan",
		res:     sqn{},
		truth:   true,
		comment: "greater since beyond window size",
	},
	{
		n1:      sqn{1000, 2048, 64},
		n2:      sqn{1100, 2048, 64},
		action:  "greaterThan",
		res:     sqn{},
		truth:   true,
		comment: "smaller except greater because beyond window size",
	},
	{
		n1:      sqn{1100, 2048, 64},
		n2:      sqn{1000, 2048, 64},
		action:  "greaterThan",
		res:     sqn{},
		truth:   true,
		comment: "greater and beyond window size",
	},
	{
		n1:      sqn{1, 2048, 64},
		n2:      sqn{2046, 2048, 64},
		action:  "greaterThan",
		res:     sqn{},
		truth:   true,
		comment: "greater by rollover",
	},
	{
		n1:      sqn{5, 2048, 64},
		n2:      sqn{10, 2048, 64},
		action:  "greaterThan",
		res:     sqn{},
		truth:   false,
		comment: "lessthan",
	},
	{
		n1:      sqn{2047, 2048, 64},
		n2:      sqn{0, 2048, 64},
		action:  "greaterThan",
		res:     sqn{},
		truth:   false,
		comment: "lessthan by rollover",
	},
	{
		n1:      sqn{20, 32, 32},
		n2:      sqn{10, 32, 32},
		action:  "greaterThan",
		res:     sqn{},
		truth:   true,
		comment: "always greater with window==limit",
	},
	{
		n1:      sqn{10, 32, 32},
		n2:      sqn{30, 32, 32},
		action:  "greaterThan",
		res:     sqn{},
		truth:   true,
		comment: "always greater with window==limit",
	},
	{
		n1:      sqn{1, 32, 32},
		n2:      sqn{31, 32, 32},
		action:  "greaterThan",
		res:     sqn{},
		truth:   true,
		comment: "always greater with window==limit",
	},
	{
		n1:      sqn{31, 32, 32},
		n2:      sqn{1, 32, 32},
		action:  "greaterThan",
		res:     sqn{},
		truth:   true,
		comment: "always greater with window==limit",
	},
	{
		n1:      sqn{212, 2048, 64},
		n2:      sqn{212, 2048, 64},
		action:  "lessThan",
		res:     sqn{},
		truth:   false,
		comment: "equal",
	},
	{
		n1:      sqn{100, 2048, 64},
		n2:      sqn{90, 2048, 64},
		action:  "lessThan",
		res:     sqn{},
		truth:   false,
		comment: "greater",
	},
	{
		n1:      sqn{2000, 2048, 64},
		n2:      sqn{2040, 2048, 64},
		action:  "lessThan",
		res:     sqn{},
		truth:   true,
		comment: "lessthan",
	},
	{
		n1:      sqn{1000, 2048, 64},
		n2:      sqn{2000, 2048, 64},
		action:  "lessThan",
		res:     sqn{},
		truth:   false,
		comment: "lessthan except beyond window",
	},
	{
		n1:      sqn{2040, 2048, 64},
		n2:      sqn{2, 2048, 64},
		action:  "lessThan",
		res:     sqn{},
		truth:   true,
		comment: "lessthan by rollover",
	},
	{
		n1:      sqn{2, 2048, 64},
		n2:      sqn{2040, 2048, 64},
		action:  "lessThan",
		res:     sqn{},
		truth:   false,
		comment: "greater by rollover",
	},
	{
		n1:      sqn{212, 2048, 64},
		n2:      sqn{213, 2048, 64},
		action:  "lessThan",
		res:     sqn{},
		truth:   true,
		comment: "one less than",
	},
	{
		n1:      sqn{212, 2048, 64},
		n2:      sqn{211, 2048, 64},
		action:  "lessThan",
		res:     sqn{},
		truth:   false,
		comment: "one more than",
	},
}

func TestSQN(t *testing.T) {
	// Testing newSQN function
	constructorPanicTest := func(num, limit, window int, comment string) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("newSQN: failed to panic:", num, limit, window, comment)
			}
		}()
		_ = newSQN(num, limit, window)
	}
	constructorPanicTest(-1, 512, 32, "sqn num must not be allowed to be negitive")
	constructorPanicTest(512, 512, 32, "sqn num must be less than address limit")
	constructorPanicTest(42, 64, 128, "window must be less than or equal to address limit")

	// Testing sqn methods
	for _, testCase := range sqnTests {
		n1 := newSQN(testCase.n1.num, testCase.n1.limit, testCase.n1.window)
		n2 := newSQN(testCase.n2.num, testCase.n2.limit, testCase.n2.window)
		switch testCase.action {
		case "add":
			out := n1.add(n2)
			pass := out.num == testCase.res.num &&
				out.limit == testCase.res.limit &&
				out.window == testCase.res.window
			if !pass {
				t.Error("sqn: add:", testCase.comment, testCase)
			}
		case "subtract":
			out := n1.subtract(n2)
			pass := out.num == testCase.res.num &&
				out.limit == testCase.res.limit &&
				out.window == testCase.res.window
			if !pass {
				t.Error("sqn: subtract:", testCase.comment, testCase)
			}
		case "equalTo":
			out := n1.equalTo(n2)
			pass := out == testCase.truth
			if !pass {
				t.Error("sqn: equalTo:", testCase.comment, testCase)
			}
		case "greaterThan":
			out := n1.greaterThan(n2)
			pass := out == testCase.truth
			if !pass {
				t.Error("sqn: greaterThan:", testCase.comment, testCase)
			}
		case "lessThan":
			out := n1.lessThan(n2)
			pass := out == testCase.truth
			if !pass {
				t.Error("sqn: lessThan:", testCase.comment, testCase)
			}
		default:
			t.Error("sqn: test case undefined:", testCase.comment, testCase.action)
		}
	}
}
