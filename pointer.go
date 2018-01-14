// Package jsonpointer implements IETF rfc6901
// JSON Pointers are a string syntax for
// identifying a specific value within a JavaScript Object Notation
// (JSON) document [RFC4627].  JSON Pointer is intended to be easily
// expressed in JSON string values as well as Uniform Resource
// Identifier (URI) [RFC3986] fragment identifiers.
//
// this package is intended to work like net/url from the go
// standard library
package jsonpointer

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Parse parses str into a Pointer structure.
// str may be a pointer or a url string.
// If a url string, Parse will use the URL's fragment component
// (the bit after the '#' symbol)
func Parse(str string) (Pointer, error) {
	// fast paths that skip url parse step
	if len(str) == 0 || str == "#" {
		return Pointer{}, nil
	} else if str[0] == '/' {
		return parse(str)
	}

	u, err := url.Parse(str)
	if err != nil {
		return nil, err
	}
	return parse(u.Fragment)
}

// The ABNF syntax of a JSON Pointer is:
// json-pointer    = *( "/" reference-token )
// reference-token = *( unescaped / escaped )
// unescaped       = %x00-2E / %x30-7D / %x7F-10FFFF
//    ; %x2F ('/') and %x7E ('~') are excluded from 'unescaped'
// escaped         = "~" ( "0" / "1" )
//   ; representing '~' and '/', respectively
func parse(str string) (Pointer, error) {
	if len(str) == 0 {
		return Pointer{}, nil
	}

	if str[0] != '/' {
		return nil, fmt.Errorf("non-empty references must begin with a '/' character")
	}
	str = str[1:]

	toks := strings.Split(str, separator)
	for i, t := range toks {
		toks[i] = unescapeToken(t)
	}
	return Pointer(toks), nil
}

// Pointer represents a parsed JSON pointer
type Pointer []string

// String implements the stringer interface for Pointer,
// giving the escaped string
func (p Pointer) String() (str string) {
	for _, tok := range p {
		str += "/" + escapeToken(tok)
	}
	return
}

// Eval evaluates a json pointer against a given root JSON document
// Evaluation of a JSON Pointer begins with a reference to the root
// value of a JSON document and completes with a reference to some value
// within the document.  Each reference token in the JSON Pointer is
// evaluated sequentially.
func (p Pointer) Eval(data interface{}) (result interface{}, err error) {
	result = data
	for _, tok := range p {
		if result, err = p.evalToken(tok, result); err != nil {
			return nil, err
		}
	}
	return
}

// Evaluation of each reference token begins by decoding any escaped
// character sequence.  This is performed by first transforming any
// occurrence of the sequence '~1' to '/', and then transforming any
// occurrence of the sequence '~0' to '~'.  By performing the
// substitutions in this order, an implementation avoids the error of
// turning '~01' first into '~1' and then into '/', which would be
// incorrect (the string '~01' correctly becomes '~1' after
// transformation).
// The reference token then modifies which value is referenced according
// to the following scheme:
func (p Pointer) evalToken(tok string, data interface{}) (interface{}, error) {
	switch ch := data.(type) {
	case map[string]interface{}:
		return ch[tok], nil
	case []interface{}:
		i, err := strconv.Atoi(tok)
		if err != nil {
			return nil, fmt.Errorf("invalid array index: %s", tok)
		}
		if i >= len(ch) {
			return nil, fmt.Errorf("index %d exceeds array length of %d", i, len(ch))
		}
		return ch[i], nil
	default:
		return nil, fmt.Errorf("invalid JSON pointer: %s", p.String())
	}
}

const (
	separator        = "/"
	escapedSeparator = "~1"
	tilde            = "~"
	escapedTilde     = "~0"
)

func unescapeToken(tok string) string {
	tok = strings.Replace(tok, escapedSeparator, separator, -1)
	return strings.Replace(tok, escapedTilde, tilde, -1)
}

func escapeToken(tok string) string {
	tok = strings.Replace(tok, escapedTilde, escapedTilde, -1)
	return strings.Replace(tok, separator, escapedSeparator, -1)
}
