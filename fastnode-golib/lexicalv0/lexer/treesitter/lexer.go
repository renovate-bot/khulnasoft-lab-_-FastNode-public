package treesitter

import (
	"log"
	"sync/atomic"

	sitter "github.com/kiteco/go-tree-sitter"
	"github.com/khulnasoft-lab/fastnode/fastnode-golib/errors"
)

const (
	// EndSymbolIdx is the index in the vocab for the treesitter internal "end" symbol
	EndSymbolIdx = 0
	// EndSymbolName is the name for the tree sitter internal "end" symbol
	EndSymbolName = "end"
)

// ErrorCount ...
var ErrorCount int64

// Token is a lexical token generated by the tree-sitter parser
type Token struct {
	Symbol     int
	SymbolName string
	Start      int
	End        int
	Lit        string
}

// Lex parses the source code in buf for the provided language and returns
// the tokens.
func Lex(buf []byte, lang *sitter.Language, tokenExtractor func([]byte, *sitter.Parser, *sitter.Tree) ([]Token, error)) (tokens []Token, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println(r)
			err = errors.Errorf("error lexing: %+v", r)
			atomic.AddInt64(&ErrorCount, 1)
		}
	}()

	p := sitter.NewParser()
	defer p.Close()
	p.SetLanguage(lang)
	tree := p.Parse(buf)
	defer tree.Close()

	return tokenExtractor(buf, p, tree)
}
