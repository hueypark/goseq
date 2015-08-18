//line grammer.y:6
package parse

import __yyfmt__ "fmt"

//line grammer.y:6
import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/scanner"
)

var DualRunes = map[string]int{
	"--": DOUBLEDASH,
	"-":  DASH,
	"=":  EQUAL,

	">>":  DOUBLEANGR,
	">":   ANGR,
	"/>":  SLASHANGR,
	"\\>": BACKSLASHANGR,
}

//line grammer.y:31
type yySymType struct {
	yys         int
	nodeList    *NodeList
	node        Node
	arrow       ArrowType
	arrowStem   ArrowStemType
	arrowHead   ArrowHeadType
	noteAlign   NoteAlignment
	dividerType GapType

	sval string
}

const K_TITLE = 57346
const K_PARTICIPANT = 57347
const K_NOTE = 57348
const K_LEFT = 57349
const K_RIGHT = 57350
const K_OVER = 57351
const K_OF = 57352
const K_HORIZONTAL = 57353
const K_GAP = 57354
const K_LINE = 57355
const K_FRAME = 57356
const DASH = 57357
const DOUBLEDASH = 57358
const EQUAL = 57359
const ANGR = 57360
const DOUBLEANGR = 57361
const BACKSLASHANGR = 57362
const SLASHANGR = 57363
const MESSAGE = 57364
const IDENT = 57365

var yyToknames = []string{
	"K_TITLE",
	"K_PARTICIPANT",
	"K_NOTE",
	"K_LEFT",
	"K_RIGHT",
	"K_OVER",
	"K_OF",
	"K_HORIZONTAL",
	"K_GAP",
	"K_LINE",
	"K_FRAME",
	"DASH",
	"DOUBLEDASH",
	"EQUAL",
	"ANGR",
	"DOUBLEANGR",
	"BACKSLASHANGR",
	"SLASHANGR",
	"MESSAGE",
	"IDENT",
}
var yyStatenames = []string{}

const yyEofCode = 1
const yyErrCode = 2
const yyMaxDepth = 200

//line grammer.y:164

// Manages the lexer as well as the current diagram being parsed
type parseState struct {
	S     scanner.Scanner
	err   error
	atEof bool
	//diagram     *Diagram
	procInstrs []string
	nodeList   *NodeList
}

func newParseState(src io.Reader, filename string) *parseState {
	ps := &parseState{}
	ps.S.Init(src)
	ps.S.Position.Filename = filename
	//    ps.diagram = &Diagram{}

	return ps
}

func (ps *parseState) Lex(lval *yySymType) int {
	if ps.atEof {
		return 0
	}
	for {
		tok := ps.S.Scan()
		switch tok {
		case scanner.EOF:
			ps.atEof = true
			return 0
		case '#':
			ps.scanComment()
		case ':':
			return ps.scanMessage(lval)
		case '-', '>', '*', '=', '/', '\\':
			if res, isTok := ps.handleDoubleRune(tok); isTok {
				return res
			} else {
				ps.Error("Invalid token: " + scanner.TokenString(tok))
			}
		case scanner.Ident:
			return ps.scanKeywordOrIdent(lval)
		default:
			ps.Error("Invalid token: " + scanner.TokenString(tok))
		}
	}
}

func (ps *parseState) handleDoubleRune(firstRune rune) (int, bool) {
	nextRune := ps.S.Peek()

	// Try the double rune
	if nextRune != scanner.EOF {
		tokStr := string(firstRune) + string(nextRune)
		if tok, hasTok := DualRunes[tokStr]; hasTok {
			ps.NextRune()
			return tok, true
		}
	}

	// Try the single rune
	tokStr := string(firstRune)
	if tok, hasTok := DualRunes[tokStr]; hasTok {
		return tok, true
	}

	return 0, false
}

func (ps *parseState) scanKeywordOrIdent(lval *yySymType) int {
	tokVal := ps.S.TokenText()
	switch strings.ToLower(tokVal) {
	case "title":
		return K_TITLE
	case "participant":
		return K_PARTICIPANT
	case "note":
		return K_NOTE
	case "left":
		return K_LEFT
	case "right":
		return K_RIGHT
	case "over":
		return K_OVER
	case "of":
		return K_OF
	case "gap":
		return K_GAP
	case "frame":
		return K_FRAME
	case "line":
		return K_LINE
	case "horizontal":
		return K_HORIZONTAL
	default:
		lval.sval = tokVal
		return IDENT
	}
}

// Scans a message.  A message is all characters up to the new line
func (ps *parseState) scanMessage(lval *yySymType) int {
	buf := new(bytes.Buffer)
	r := ps.NextRune()
	for (r != '\n') && (r != scanner.EOF) {
		if r == '\\' {
			nr := ps.NextRune()
			switch nr {
			case 'n':
				buf.WriteRune('\n')
			case '\\':
				buf.WriteRune('\\')
			default:
				ps.Error("Invalid backslash escape: \\" + string(nr))
			}
		} else {
			buf.WriteRune(r)
		}
		r = ps.NextRune()
	}

	lval.sval = strings.TrimSpace(buf.String())
	return MESSAGE
}

// Scans a comment.  This ignores all characters up to the new line.
func (ps *parseState) scanComment() {
	var buf *bytes.Buffer

	r := ps.NextRune()
	if r == '!' {
		// This starts a processor instruction
		buf = new(bytes.Buffer)
		r = ps.NextRune()
	}

	for (r != '\n') && (r != scanner.EOF) {
		if buf != nil {
			buf.WriteRune(r)
		}
		r = ps.NextRune()
	}

	if buf != nil {
		ps.procInstrs = append(ps.procInstrs, strings.TrimSpace(buf.String()))
	}
}

func (ps *parseState) NextRune() rune {
	if ps.atEof {
		return scanner.EOF
	}

	r := ps.S.Next()
	if r == scanner.EOF {
		ps.atEof = true
	}

	return r
}

func (ps *parseState) Error(err string) {
	errMsg := fmt.Sprintf("%s:%d: %s", ps.S.Position.Filename, ps.S.Position.Line, err)
	ps.err = errors.New(errMsg)
}

func Parse(reader io.Reader, filename string) (*NodeList, error) {
	ps := newParseState(reader, filename)
	yyParse(ps)

	// Add processing instructions to the start of the node list
	for i := len(ps.procInstrs) - 1; i >= 0; i-- {
		instrParts := strings.SplitN(ps.procInstrs[i], " ", 2)
		name, value := strings.TrimSpace(instrParts[0]), strings.TrimSpace(instrParts[1])
		ps.nodeList = &NodeList{&ProcessInstructionNode{name, value}, ps.nodeList}
	}

	if ps.err != nil {
		return nil, ps.err
	} else {
		return ps.nodeList, nil
	}
}

//line yacctab:1
var yyExca = []int{
	-1, 1,
	1, -1,
	-2, 0,
}

const yyNprod = 30
const yyPrivate = 57344

var yyTokenNames []string
var yyStates []string

const yyLast = 42

var yyAct = []int{

	9, 10, 12, 33, 34, 35, 36, 13, 39, 37,
	31, 16, 42, 41, 40, 30, 15, 38, 26, 11,
	19, 20, 21, 27, 28, 29, 23, 24, 25, 2,
	22, 32, 18, 14, 17, 8, 7, 6, 5, 4,
	3, 1,
}
var yyPact = []int{

	-4, -1000, -1000, -4, -1000, -1000, -1000, -1000, -1000, -6,
	-12, 5, 19, 11, -1000, -1000, -7, -13, -15, -1000,
	-1000, -1000, -14, 7, -2, -1000, -8, -1000, -1000, -1000,
	-1000, -9, -1000, -1000, -1000, -1000, -1000, -10, -1000, -1000,
	-1000, -1000, -1000,
}
var yyPgo = []int{

	0, 41, 29, 40, 39, 38, 37, 36, 35, 34,
	32, 31, 30, 18,
}
var yyR1 = []int{

	0, 1, 2, 2, 3, 3, 3, 3, 3, 4,
	5, 5, 6, 7, 8, 8, 13, 13, 13, 12,
	12, 12, 9, 10, 10, 10, 11, 11, 11, 11,
}
var yyR2 = []int{

	0, 1, 0, 2, 1, 1, 1, 1, 1, 2,
	2, 3, 4, 4, 2, 3, 1, 1, 1, 2,
	2, 1, 2, 1, 1, 1, 1, 1, 1, 1,
}
var yyChk = []int{

	-1000, -1, -2, -3, -4, -5, -6, -7, -8, 4,
	5, 23, 6, 11, -2, 22, 23, -9, -10, 15,
	16, 17, -12, 7, 8, 9, -13, 12, 13, 14,
	22, 23, -11, 18, 19, 20, 21, 23, 10, 10,
	22, 22, 22,
}
var yyDef = []int{

	2, -2, 1, 2, 4, 5, 6, 7, 8, 0,
	0, 0, 0, 0, 3, 9, 10, 0, 0, 23,
	24, 25, 0, 0, 0, 21, 14, 16, 17, 18,
	11, 0, 22, 26, 27, 28, 29, 0, 19, 20,
	15, 12, 13,
}
var yyTok1 = []int{

	1,
}
var yyTok2 = []int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
	22, 23,
}
var yyTok3 = []int{
	0,
}

//line yaccpar:1

/*	parser for yacc output	*/

var yyDebug = 0

type yyLexer interface {
	Lex(lval *yySymType) int
	Error(s string)
}

const yyFlag = -1000

func yyTokname(c int) string {
	// 4 is TOKSTART above
	if c >= 4 && c-4 < len(yyToknames) {
		if yyToknames[c-4] != "" {
			return yyToknames[c-4]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func yyStatname(s int) string {
	if s >= 0 && s < len(yyStatenames) {
		if yyStatenames[s] != "" {
			return yyStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func yylex1(lex yyLexer, lval *yySymType) int {
	c := 0
	char := lex.Lex(lval)
	if char <= 0 {
		c = yyTok1[0]
		goto out
	}
	if char < len(yyTok1) {
		c = yyTok1[char]
		goto out
	}
	if char >= yyPrivate {
		if char < yyPrivate+len(yyTok2) {
			c = yyTok2[char-yyPrivate]
			goto out
		}
	}
	for i := 0; i < len(yyTok3); i += 2 {
		c = yyTok3[i+0]
		if c == char {
			c = yyTok3[i+1]
			goto out
		}
	}

out:
	if c == 0 {
		c = yyTok2[1] /* unknown char */
	}
	if yyDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", yyTokname(c), uint(char))
	}
	return c
}

func yyParse(yylex yyLexer) int {
	var yyn int
	var yylval yySymType
	var yyVAL yySymType
	yyS := make([]yySymType, yyMaxDepth)

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	yystate := 0
	yychar := -1
	yyp := -1
	goto yystack

ret0:
	return 0

ret1:
	return 1

yystack:
	/* put a state and value onto the stack */
	if yyDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", yyTokname(yychar), yyStatname(yystate))
	}

	yyp++
	if yyp >= len(yyS) {
		nyys := make([]yySymType, len(yyS)*2)
		copy(nyys, yyS)
		yyS = nyys
	}
	yyS[yyp] = yyVAL
	yyS[yyp].yys = yystate

yynewstate:
	yyn = yyPact[yystate]
	if yyn <= yyFlag {
		goto yydefault /* simple state */
	}
	if yychar < 0 {
		yychar = yylex1(yylex, &yylval)
	}
	yyn += yychar
	if yyn < 0 || yyn >= yyLast {
		goto yydefault
	}
	yyn = yyAct[yyn]
	if yyChk[yyn] == yychar { /* valid shift */
		yychar = -1
		yyVAL = yylval
		yystate = yyn
		if Errflag > 0 {
			Errflag--
		}
		goto yystack
	}

yydefault:
	/* default state action */
	yyn = yyDef[yystate]
	if yyn == -2 {
		if yychar < 0 {
			yychar = yylex1(yylex, &yylval)
		}

		/* look through exception table */
		xi := 0
		for {
			if yyExca[xi+0] == -1 && yyExca[xi+1] == yystate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			yyn = yyExca[xi+0]
			if yyn < 0 || yyn == yychar {
				break
			}
		}
		yyn = yyExca[xi+1]
		if yyn < 0 {
			goto ret0
		}
	}
	if yyn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			yylex.Error("syntax error")
			Nerrs++
			if yyDebug >= 1 {
				__yyfmt__.Printf("%s", yyStatname(yystate))
				__yyfmt__.Printf(" saw %s\n", yyTokname(yychar))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for yyp >= 0 {
				yyn = yyPact[yyS[yyp].yys] + yyErrCode
				if yyn >= 0 && yyn < yyLast {
					yystate = yyAct[yyn] /* simulate a shift of "error" */
					if yyChk[yystate] == yyErrCode {
						goto yystack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if yyDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", yyS[yyp].yys)
				}
				yyp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if yyDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", yyTokname(yychar))
			}
			if yychar == yyEofCode {
				goto ret1
			}
			yychar = -1
			goto yynewstate /* try again in the same state */
		}
	}

	/* reduction by production yyn */
	if yyDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", yyn, yyStatname(yystate))
	}

	yynt := yyn
	yypt := yyp
	_ = yypt // guard against "declared and not used"

	yyp -= yyR2[yyn]
	yyVAL = yyS[yyp+1]

	/* consult goto table to find next state */
	yyn = yyR1[yyn]
	yyg := yyPgo[yyn]
	yyj := yyg + yyS[yyp].yys + 1

	if yyj >= yyLast {
		yystate = yyAct[yyg]
	} else {
		yystate = yyAct[yyj]
		if yyChk[yystate] != -yyn {
			yystate = yyAct[yyg]
		}
	}
	// dummy call; replaced with literal code
	switch yynt {

	case 1:
		//line grammer.y:66
		{
			yylex.(*parseState).nodeList = yyS[yypt-0].nodeList
		}
	case 2:
		//line grammer.y:73
		{
			yyVAL.nodeList = nil
		}
	case 3:
		//line grammer.y:77
		{
			yyVAL.nodeList = &NodeList{yyS[yypt-1].node, yyS[yypt-0].nodeList}
		}
	case 4:
		yyVAL.node = yyS[yypt-0].node
	case 5:
		yyVAL.node = yyS[yypt-0].node
	case 6:
		yyVAL.node = yyS[yypt-0].node
	case 7:
		yyVAL.node = yyS[yypt-0].node
	case 8:
		yyVAL.node = yyS[yypt-0].node
	case 9:
		//line grammer.y:92
		{
			yyVAL.node = &TitleNode{yyS[yypt-0].sval}
		}
	case 10:
		//line grammer.y:99
		{
			yyVAL.node = &ActorNode{yyS[yypt-0].sval, false, ""}
		}
	case 11:
		//line grammer.y:103
		{
			yyVAL.node = &ActorNode{yyS[yypt-1].sval, true, yyS[yypt-0].sval}
		}
	case 12:
		//line grammer.y:110
		{
			yyVAL.node = &ActionNode{yyS[yypt-3].sval, yyS[yypt-1].sval, yyS[yypt-2].arrow, yyS[yypt-0].sval}
		}
	case 13:
		//line grammer.y:117
		{
			yyVAL.node = &NoteNode{yyS[yypt-1].sval, yyS[yypt-2].noteAlign, yyS[yypt-0].sval}
		}
	case 14:
		//line grammer.y:124
		{
			yyVAL.node = &GapNode{yyS[yypt-0].dividerType, ""}
		}
	case 15:
		//line grammer.y:128
		{
			yyVAL.node = &GapNode{yyS[yypt-1].dividerType, yyS[yypt-0].sval}
		}
	case 16:
		//line grammer.y:134
		{
			yyVAL.dividerType = EMPTY_GAP
		}
	case 17:
		//line grammer.y:135
		{
			yyVAL.dividerType = LINE_GAP
		}
	case 18:
		//line grammer.y:136
		{
			yyVAL.dividerType = FRAME_GAP
		}
	case 19:
		//line grammer.y:140
		{
			yyVAL.noteAlign = LEFT_NOTE_ALIGNMENT
		}
	case 20:
		//line grammer.y:141
		{
			yyVAL.noteAlign = RIGHT_NOTE_ALIGNMENT
		}
	case 21:
		//line grammer.y:142
		{
			yyVAL.noteAlign = OVER_NOTE_ALIGNMENT
		}
	case 22:
		//line grammer.y:147
		{
			yyVAL.arrow = ArrowType{yyS[yypt-1].arrowStem, yyS[yypt-0].arrowHead}
		}
	case 23:
		//line grammer.y:153
		{
			yyVAL.arrowStem = SOLID_ARROW_STEM
		}
	case 24:
		//line grammer.y:154
		{
			yyVAL.arrowStem = DASHED_ARROW_STEM
		}
	case 25:
		//line grammer.y:155
		{
			yyVAL.arrowStem = THICK_ARROW_STEM
		}
	case 26:
		//line grammer.y:159
		{
			yyVAL.arrowHead = SOLID_ARROW_HEAD
		}
	case 27:
		//line grammer.y:160
		{
			yyVAL.arrowHead = OPEN_ARROW_HEAD
		}
	case 28:
		//line grammer.y:161
		{
			yyVAL.arrowHead = BARBED_ARROW_HEAD
		}
	case 29:
		//line grammer.y:162
		{
			yyVAL.arrowHead = LOWER_BARBED_ARROW_HEAD
		}
	}
	goto yystack /* stack new state and value */
}