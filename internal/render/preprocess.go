package render

import (
	"bytes"
	"unicode"
)

// preprocessMarkdown fixes two markdown rendering issues before goldmark sees the source:
//
//  1. Single ~ used as a range indicator (e.g. 80~90) is escaped to prevent
//     goldmark's GFM strikethrough extension from treating it as a delimiter.
//
//  2. Emphasis markers (**/__/*/_) that immediately follow closing punctuation
//     (e.g. **text(paren)**다음단어) fail CommonMark's right-flanking rule when
//     the following character is a CJK letter.  A zero-width space (U+200B) is
//     inserted before the delimiter so goldmark recognises it as a closer.
func preprocessMarkdown(src []byte) []byte {
	lines := bytes.SplitAfter(src, []byte("\n"))
	var out bytes.Buffer
	inFence := false

	for _, line := range lines {
		stripped := bytes.TrimLeft(line, " \t")
		// Fenced code block boundary: ``` or ~~~
		if len(stripped) >= 3 &&
			(stripped[0] == '`' || stripped[0] == '~') &&
			stripped[0] == stripped[1] && stripped[1] == stripped[2] {
			inFence = !inFence
			out.Write(line)
			continue
		}
		if inFence {
			out.Write(line)
			continue
		}
		out.Write(processInlineLine(line))
	}
	return out.Bytes()
}

func processInlineLine(line []byte) []byte {
	rs := []rune(string(line))
	out := make([]rune, 0, len(rs)+8)
	i := 0

	for i < len(rs) {
		r := rs[i]

		// Inline code span — pass through unchanged so we don't mangle code content.
		if r == '`' {
			n := 0
			for i+n < len(rs) && rs[i+n] == '`' {
				n++
			}
			out = append(out, rs[i:i+n]...)
			i += n
			for i < len(rs) {
				if rs[i] == '`' {
					m := 0
					for i+m < len(rs) && rs[i+m] == '`' {
						m++
					}
					out = append(out, rs[i:i+m]...)
					i += m
					if m == n {
						break
					}
				} else {
					out = append(out, rs[i])
					i++
				}
			}
			continue
		}

		// Tilde: ~~ kept as-is (strikethrough), single ~ escaped.
		if r == '~' {
			if i+1 < len(rs) && rs[i+1] == '~' {
				out = append(out, '~', '~')
				i += 2
			} else {
				out = append(out, '\\', '~')
				i++
			}
			continue
		}

		// Emphasis delimiter run: insert U+200B before the run when the
		// preceding character is punctuation or a symbol so that goldmark's
		// right-flanking check passes for CJK-adjacent emphasis.
		if r == '*' || r == '_' {
			n := 0
			for i+n < len(rs) && rs[i+n] == r {
				n++
			}
			if i > 0 && (unicode.IsPunct(rs[i-1]) || unicode.IsSymbol(rs[i-1])) {
				out = append(out, '​')
			}
			out = append(out, rs[i:i+n]...)
			i += n
			continue
		}

		out = append(out, r)
		i++
	}

	return []byte(string(out))
}
