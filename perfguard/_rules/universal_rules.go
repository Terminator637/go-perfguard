package gorules

import (
	"github.com/quasilyte/go-ruleguard/dsl"
)

// Universal rules are shared in both `lint` and `optimize` modes.
//
// By default, all rules trigger on every successful match.
// For most optimization rules, it's better to set one of the
// tags to limit its scope of application.
//
// There are two special tags for this: o1 and o2.
// o1 requires that heat level for this line is not zero
// o2 requires that heat level for this line is 5 (max level)
//
// Use o2 for rules that should be applied carefully.
// This is usually the case when optimized code is more verbose
// or generally less pretty.
//
// Lint mode ignores o1 and o2 tags completely.

//doc:summary Detects unoptimal strings/bytes case-insensitive comparison
//doc:tags    o1
//doc:before  strings.ToLower(x) == strings.ToLower(y)
//doc:after   strings.EqualFold(x, y)
func equalFold(m dsl.Matcher) {
	// string == patterns
	m.Match(
		`strings.ToLower($x) == $y`,
		`strings.ToLower($x) == strings.ToLower($y)`,
		`$x == strings.ToLower($y)`,
		`strings.ToUpper($x) == $y`,
		`strings.ToUpper($x) == strings.ToUpper($y)`,
		`$x == strings.ToUpper($y)`).
		Where(m["x"].Pure && m["y"].Pure && m["x"].Text != m["y"].Text).
		Suggest(`strings.EqualFold($x, $y)`)

	// string != patterns
	m.Match(
		`strings.ToLower($x) != $y`,
		`strings.ToLower($x) != strings.ToLower($y)`,
		`$x != strings.ToLower($y)`,
		`strings.ToUpper($x) != $y`,
		`strings.ToUpper($x) != strings.ToUpper($y)`,
		`$x != strings.ToUpper($y)`).
		Where(m["x"].Pure && m["y"].Pure && m["x"].Text != m["y"].Text).
		Suggest(`!strings.EqualFold($x, $y)`)

	// bytes.Equal patterns
	m.Match(
		`bytes.Equal(bytes.ToLower($x), $y)`,
		`bytes.Equal(bytes.ToLower($x), bytes.ToLower($y))`,
		`bytes.Equal($x, bytes.ToLower($y))`,
		`bytes.Equal(bytes.ToUpper($x), $y)`,
		`bytes.Equal(bytes.ToUpper($x), bytes.ToUpper($y))`,
		`bytes.Equal($x, bytes.ToUpper($y))`).
		Where(m["x"].Pure && m["y"].Pure && m["x"].Text != m["y"].Text).
		Suggest(`bytes.EqualFold($x, $y)`)
}

//doc:summary Detects redundant fmt.Sprint calls
//doc:tags    o1
func redundantSprint(m dsl.Matcher) {
	m.Match(`fmt.Sprint($x)`, `fmt.Sprintf("%s", $x)`, `fmt.Sprintf("%v", $x)`).
		Where(m["x"].Type.Implements(`fmt.Stringer`)).
		Suggest(`$x.String()`)

	m.Match(`fmt.Sprint($x)`, `fmt.Sprintf("%s", $x)`, `fmt.Sprintf("%v", $x)`).
		Where(m["x"].Type.Implements(`error`)).
		Suggest(`$x.Error()`)

	m.Match(`fmt.Sprint($x)`, `fmt.Sprintf("%s", $x)`, `fmt.Sprintf("%v", $x)`).
		Where(m["x"].Type.Is(`string`)).
		Suggest(`$x`)
}

//doc:summary Detect strings.Join usages that can be rewritten as a string concat
//doc:tags    o1
func stringsJoinConcat(m dsl.Matcher) {
	m.Match(`strings.Join([]string{$x, $y}, "")`).Suggest(`$x + $y`)
	m.Match(`strings.Join([]string{$x, $y, $z}, "")`).Suggest(`$x + $y + $z`)

	m.Match(`strings.Join([]string{$x, $y}, $glue)`).Suggest(`$x + $glue + $y`)

	m.Match(`strings.Join([]string{$x, $y, $z}, $glue)`).
		Where(m["glue"].Pure).
		Suggest(`$x + $glue + $y + $glue + $z`)
}

//doc:summary Detects sprint calls that can be rewritten as a string concat
//doc:tags    o1
//doc:before  fmt.Sprintf("%s%s", x, y)
//doc:after   x + y
func sprintConcat(m dsl.Matcher) {
	m.Match(`fmt.Sprintf("%s%s", $x, $y)`).
		Where(m["x"].Type.Is(`string`) && m["y"].Type.Is(`string`)).
		Suggest(`$x + $y`)

	m.Match(`fmt.Sprintf("%s%s", $x, $y)`).
		Where(m["x"].Type.Implements(`fmt.Stringer`) && m["y"].Type.Implements(`fmt.Stringer`)).
		Suggest(`$x.String() + $y.String()`)
}

//doc:summary Detects fmt uses that can be replaced with strconv
//doc:tags    o1
//doc:before  fmt.Sprintf("%d", i)
//doc:after   strconv.Itoa(i)
func strconv(m dsl.Matcher) {
	// Sprint(x) is basically Sprintf("%v", x), so we treat it identically.

	// The most simple cases that can be converted to Itoa.
	m.Match(`fmt.Sprintf("%d", $x)`, `fmt.Sprintf("%v", $x)`, `fmt.Sprint($x)`).
		Where(m["x"].Type.Is(`int`)).Suggest(`strconv.Itoa($x)`)

	// Patterns for int64 and uint64 go first,
	// so we don't insert unnecessary conversions by the rules below.
	m.Match(`fmt.Sprintf("%d", $x)`, `fmt.Sprintf("%v", $x)`, `fmt.Sprint($x)`).
		Where(m["x"].Type.Is(`int64`)).Suggest(`strconv.FormatInt($x, 10)`)
	m.Match(`fmt.Sprintf("%x", $x)`).
		Where(m["x"].Type.Is(`int64`)).Suggest(`strconv.FormatInt($x, 16)`)
	m.Match(`fmt.Sprintf("%d", $x)`, `fmt.Sprintf("%v", $x)`, `fmt.Sprint($x)`).
		Where(m["x"].Type.Is(`uint64`)).Suggest(`strconv.FormatUint($x, 10)`)
	m.Match(`fmt.Sprintf("%x", $x)`).
		Where(m["x"].Type.Is(`uint64`)).Suggest(`strconv.FormatUint($x, 16)`)

	m.Match(`fmt.Sprintf("%d", $x)`, `fmt.Sprintf("%v", $x)`, `fmt.Sprint($x)`).
		Where(m["x"].Type.OfKind(`int`)).Suggest(`strconv.FormatInt(int64($x), 10)`)
	m.Match(`fmt.Sprintf("%x", $x)`).
		Where(m["x"].Type.OfKind(`int`)).Suggest(`strconv.FormatInt(int64($x), 16)`)

	m.Match(`fmt.Sprintf("%d", $x)`, `fmt.Sprintf("%v", $x)`, `fmt.Sprint($x)`).
		Where(m["x"].Type.OfKind(`uint`)).Suggest(`strconv.FormatUint(uint64($x), 10)`)
	m.Match(`fmt.Sprintf("%x", $x)`).
		Where(m["x"].Type.OfKind(`uint`)).Suggest(`strconv.FormatUint(uint64($x), 16)`)
}

//doc:summary Detects cases that can benefit from append-friendly APIs
//doc:tags    o1
//doc:before  b = append(b, strconv.Itoa(v)...)
//doc:after   b = strconv.AppendInt(b, v, 10)
func appendAPI(m dsl.Matcher) {
	// append functions are generally much better than alternatives,
	// but we can only go so far with the rules.
	// Maybe it's worthwhile to implement more thorough analysis
	// that detects where append-style APIs can be used.

	// Not checking the fmt.Sprint cases and alike as they
	// should be handled by other rule.
	m.Match(`$b = append($b, strconv.Itoa($x)...)`).
		Suggest(`$b = strconv.AppendInt($b, int64($x), 10)`)
	m.Match(`$b = append($b, strconv.FormatInt($x, $base)...)`).
		Suggest(`$b = strconv.AppendInt($b, $x, $base)`)
	m.Match(`$b = append($b, strconv.FormatUint($x, $base)...)`).
		Suggest(`$b = strconv.AppendUint($b, $x, $base)`)

	m.Match(`$b = append($b, $t.Format($layout)...)`).
		Where(m["t"].Type.Is(`time.Time`) || m["t"].Type.Is(`*time.Time`)).
		Suggest(`$b = $t.AppendFormat($b, $layout)`)

	m.Match(`$b = append($b, $v.String()...)`).
		Where(m["v"].Type.Is(`big.Float`) || m["v"].Type.Is(`*big.Float`)).
		Suggest(`$b = $v.Append($b, 'g', 10)`)
	m.Match(`$b = append($b, $v.Text($format, $prec)...)`).
		Where(m["v"].Type.Is(`big.Float`) || m["v"].Type.Is(`*big.Float`)).
		Suggest(`$b = $v.Append($b, $format, $prec)`)

	m.Match(`$b = append($b, $v.String()...)`).
		Where(m["v"].Type.Is(`big.Int`) || m["v"].Type.Is(`*big.Int`)).
		Suggest(`$b = $v.Append($b, 10)`)
	m.Match(`$b = append($b, $v.Text($base)...)`).
		Where(m["v"].Type.Is(`big.Int`) || m["v"].Type.Is(`*big.Int`)).
		Suggest(`$b = $v.Append($b, $base)`)
}

//doc:summary Detects redundant conversions between string and []byte
//doc:tags    o1
//doc:before  copy(b, []byte(s))
//doc:after   copy(b, s)
func stringCopyElim(m dsl.Matcher) {
	m.Match(`copy($b, []byte($s))`).
		Where(m["s"].Type.Is(`string`)).
		Suggest(`copy($b, $s)`)

	m.Match(`append($b, []byte($s)...)`).
		Where(m["s"].Type.Is(`string`)).
		Suggest(`append($b, $s...)`)

	m.Match(`len(string($b))`).Where(m["b"].Type.Is(`[]byte`)).Suggest(`len($b)`)

	m.Match(`$re.Match([]byte($s))`).
		Where(m["re"].Type.Is(`*regexp.Regexp`) && m["s"].Type.Is(`string`)).
		Suggest(`$re.MatchString($s)`)

	m.Match(`$re.FindIndex([]byte($s))`).
		Where(m["re"].Type.Is(`*regexp.Regexp`) && m["s"].Type.Is(`string`)).
		Suggest(`$re.FindStringIndex($s)`)

	m.Match(`$re.FindAllIndex([]byte($s), $n)`).
		Where(m["re"].Type.Is(`*regexp.Regexp`) && m["s"].Type.Is(`string`)).
		Suggest(`$re.FindAllStringIndex($s, $n)`)
}

//doc:summary Detects strings.Index()-like calls that may allocate more than they should
//doc:tags    o1
//doc:before  strings.Index(string(x), y)
//doc:after   bytes.Index(x, []byte(y))
//doc:note    See Go issue for details: https://github.com/golang/go/issues/25864
func indexAlloc(m dsl.Matcher) {
	// These rules work on the observation that substr/search item
	// is usually smaller than the containing string.

	canOptimizeStrings := func(m dsl.Matcher) bool {
		return m["x"].Pure && m["y"].Pure &&
			!m["y"].Node.Is(`CallExpr`) &&
			m["x"].Type.Is(`[]byte`)
	}

	m.Match(`strings.Index(string($x), $y)`).Where(canOptimizeStrings(m)).Suggest(`bytes.Index($x, []byte($y))`)
	m.Match(`strings.Contains(string($x), $y)`).Where(canOptimizeStrings(m)).Suggest(`bytes.Contains($x, []byte($y))`)
	m.Match(`strings.HasPrefix(string($x), $y)`).Where(canOptimizeStrings(m)).Suggest(`bytes.HasPrefix($x, []byte($y))`)
	m.Match(`strings.HasSuffix(string($x), $y)`).Where(canOptimizeStrings(m)).Suggest(`bytes.HasSuffix($x, []byte($y))`)

	canOptimizeBytes := func(m dsl.Matcher) bool {
		return m["x"].Pure && m["y"].Pure &&
			!m["y"].Node.Is(`CallExpr`) &&
			m["x"].Type.Is(`string`)
	}

	m.Match(`bytes.Index([]byte($x), $y)`).Where(canOptimizeBytes(m)).Suggest(`strings.Index($x, string($y))`)
	m.Match(`bytes.Contains([]byte($x), $y)`).Where(canOptimizeBytes(m)).Suggest(`strings.Contains($x, string($y))`)
	m.Match(`bytes.HasPrefix([]byte($x), $y)`).Where(canOptimizeBytes(m)).Suggest(`strings.HasPrefix($x, string($y))`)
	m.Match(`bytes.HasSuffix([]byte($x), $y)`).Where(canOptimizeBytes(m)).Suggest(`strings.HasSuffix($x, string($y))`)
}

//doc:summary Detects WriteRune calls with rune literal argument that is single byte and reports to use WriteByte instead
//doc:tags    o1
//doc:before  w.WriteRune('\n')
//doc:after   w.WriteByte('\n')
func writeByte(m dsl.Matcher) {
	// utf8.RuneSelf:
	// characters below RuneSelf are represented as themselves in a single byte.
	const runeSelf = 0x80
	m.Match(`$w.WriteRune($c)`).
		Where(m["w"].Type.Implements("io.ByteWriter") && (m["c"].Const && m["c"].Value.Int() < runeSelf)).
		Suggest(`$w.WriteByte($c)`)
}

//doc:summary Detects slice clear loops, suggests an idiom that is recognized by the Go compiler
//doc:tags    o1
//doc:before  for i := 0; i < len(buf); i++ { buf[i] = 0 }
//doc:after   for i := range buf { buf[i] = 0 }
func sliceClear(m dsl.Matcher) {
	m.Match(`for $i := 0; $i < len($xs); $i++ { $xs[$i] = $zero }`).
		Where(m["zero"].Value.Int() == 0).
		Suggest(`for $i := range $xs { $xs[$i] = $zero }`).
		Report(`for ... { ... } => for $i := range $xs { $xs[$i] = $zero }`)
}

//doc:summary Detects expressions like []rune(s)[0] that may cause unwanted rune slice allocation
//doc:tags    o1
//doc:before  r := []rune(s)[0]
//doc:after   r, _ := utf8.DecodeRuneInString(s)
//doc:note    See Go issue for details: https://github.com/golang/go/issues/45260
func utf8DecodeRune(m dsl.Matcher) {
	// TODO: instead of File().Imports("utf8") filter we
	// want to have a way to import "utf8" package if it's not yet imported.
	// See https://github.com/quasilyte/go-ruleguard/issues/329
	// Or maybe we can run goimports (as a library?) for these cases.
	// goimports may add more diff noise though (like imports order, etc).

	m.Match(`$ch := []rune($s)[0]`).
		Where(m["s"].Type.Is(`string`) && m.File().Imports(`unicode/utf8`)).
		Suggest(`$ch, _ := utf8.DecodeRuneInString($ch)`)

	m.Match(`$ch = []rune($s)[0]`).
		Where(m["s"].Type.Is(`string`) && m.File().Imports(`unicode/utf8`)).
		Suggest(`$ch, _ = utf8.DecodeRuneInString($ch)`)

	// Without !Imports this rule will result in duplicated messages
	// for a single slice conversion.
	m.Match(`[]rune($s)[0]`).
		Where(m["s"].Type.Is(`string`) && !m.File().Imports(`unicode/utf8`)).
		Report(`use utf8.DecodeRuneInString($s) here`)
}

//doc:summary Detects fmt.Sprint(f/ln) calls which can be replaced with fmt.Fprint(f/ln)
//doc:tags    o1
//doc:before  w.Write([]byte(fmt.Sprintf("%x", 10)))
//doc:after   fmt.Fprintf(w, "%x", 10)
func fprint(m dsl.Matcher) {
	m.Match(`$w.Write([]byte(fmt.Sprint($*args)))`).
		Where(m["w"].Type.Implements("io.Writer")).
		Suggest(`fmt.Fprint($w, $args)`)

	m.Match(`$w.Write([]byte(fmt.Sprintf($*args)))`).
		Where(m["w"].Type.Implements("io.Writer")).
		Suggest(`fmt.Fprintf($w, $args)`)

	m.Match(`$w.Write([]byte(fmt.Sprintln($*args)))`).
		Where(m["w"].Type.Implements("io.Writer")).
		Suggest(`fmt.Fprintln($w, $args)`)

	m.Match(`io.WriteString($w, fmt.Sprint($*args))`).
		Suggest(`fmt.Fprint($w, $args)`)

	m.Match(`io.WriteString($w, fmt.Sprintf($*args))`).
		Suggest(`fmt.Fprintf($w, $args)`)

	m.Match(`io.WriteString($w, fmt.Sprintln($*args))`).
		Suggest(`fmt.Fprintln($w, $args)`)
}

//doc:summary Detects w.Write calls which can be replaced with w.WriteString
//doc:tags    o1
//doc:before  w.Write([]byte("foo"))
//doc:after   w.WriteString("foo")
func writeString(m dsl.Matcher) {
	m.Match(`$w.Write([]byte($s))`).
		Where(m["w"].Type.Implements("io.StringWriter") && m["s"].Type.Is(`string`)).
		Suggest("$w.WriteString($s)")
}

//doc:summary Detects w.WriteString calls which can be replaced with w.Write
//doc:tags    o1
//doc:before  w.WriteString(buf.String())
//doc:after   w.Write(buf.Bytes())
func writeBytes(m dsl.Matcher) {
	isBuffer := func(v dsl.Var) bool {
		return v.Type.Is(`bytes.Buffer`) || v.Type.Is(`*bytes.Buffer`)
	}

	m.Match(`io.WriteString($w, $buf.String())`).
		Where(isBuffer(m["buf"])).
		Suggest(`$w.Write($buf.Bytes())`)

	m.Match(`io.WriteString($w, string($buf.Bytes()))`).
		Where(isBuffer(m["buf"])).
		Suggest(`$w.Write($buf.Bytes())`)

	m.Match(`$w.WriteString($buf.String())`).
		Where(m["w"].Type.Implements("io.Writer") && isBuffer(m["buf"])).
		Suggest(`$w.Write($buf.Bytes())`)
}
