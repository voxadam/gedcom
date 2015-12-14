package gedcom

import (
	"errors"
	"io"
	"strconv"
)

type line struct {
	level  uint64
	xrefID string
	tag    string
	value  string
}

type Reader struct {
	t tokeniser

	peeked bool
	line   line
	err    error

	hadHeader, hadRecord bool
}

func NewReader(r io.Reader) *Reader {
	return &Reader{
		r: newTokeniser(r),
	}
}

func (r *Reader) readLine() {
	if r.err != nil {
		return
	}
	var t token
	t, r.err = r.t.GetToken()
	if r.err != nil {
		return
	}
	if t.typ != tokenLevel {
		r.err = ErrNotLevel
		return
	}
	r.line.level, r.err = strconv.ParseUint(t.data, 10, 64)
	if r.err != nil {
		return
	}
	t, r.err = r.t.GetToken()
	if r.err != nil {
		return
	}
	if t.typ == tokenXref {
		line.xrefID = t.data
		t, r.err = r.t.GetToken()
		if r.err != nil {
			return
		}
	}
	if token.typ != tokenTag {
		r.err = ErrNotType
		return
	}
	r.line.tag = token.data
	t, r.err = r.t.GetToken()
	if r.err != nil {
		return
	}
	if t.typ == tokenEndLine {
		return
	}
	if t.typ != tokenLine {
		r.err = ErrNotLine
		return
	}
	r.line.value = t.data
}

func (r *Reader) Record() (Record, error) {
	if !r.peeked {
		r.readLine()
		r.peeked = true
	}
	if r.err != nil {
		return nil, err
	}
	if !r.hadHeader {
		if r.line.tag != "HEAD" {
			r.peeked = false
			return nil, ErrNoHeader
		}
		r.hadHeader = true
	} else if !r.hadRecord {
		switch r.line.tag {
		case "FAM", "INDI", "OBJE", "NOTE", "REPO", "SOUR", "SUBN":
			r.hadRecord = true
		case "TRLR":
			r.peeked = false
			return nil, ErrNoRecords
		}
	} else if r.line.tag == "TRLR" {
		r.peeked = false
		return RecordTrailer{}, nil
	}

	lines := make([]line, 1, 32)
	lines[0] = r.line
	var lastlevel = 0
	for {
		if r.err != nil {
			return nil, r.err
		}
		if r.line.level > lastlevel+1 {
			return nil, ErrInvalidLevel
		}
		lastlevel = r.line.level
		lines = append(lines, r.line)
		r.readLine()
		if r.line.level == 0 {
			plines := parseLines(lines)
			switch lines[0].tag {
			case "HEAD":
				return parseHeader(plines)
			case "SUBM":
				return parseSubmitter(plines)
			case "FAM":
				return parseFamily(plines)
			case "INDI":
				return parseIndividual(plines)
			case "OBJE":
				return parseObject(plines)
			case "NOTE":
				return parseNote(plines)
			case "REPO":
				return parseRepository(plines)
			case "SOUR":
				return parseSource(plines)
			case "SUBN":
				return parseSubmission(plines)
			default:
				if lines[0][0] == "_" {
					return plines, nil
				}
				return plines, ErrUnknownTag
			}
			return r, nil
		}
	}

}

// Errors
var (
	ErrNoHeader     = errors.New("no header")
	ErrNoRecords    = errors.New("no records")
	ErrUnknownTag   = errors.New("unknown tag name")
	ErrInvalidLevel = errors.New("invalid level")
)