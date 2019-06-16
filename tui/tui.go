package tui

import (
	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
	log "github.com/sirupsen/logrus"
	"time"
	"unicode/utf8"
)

var (
	termX        int
	termY        int
	termW        int
	termH        int
	colorDefault termbox.Attribute
	inputBox     = &InputBox{}
	messageBox   = &MessageBox{}
	eventChan    = make(chan termbox.Event)
	inputChan    = make(chan []byte)
	outputChan   = make(chan []byte)
	inputCtlChan = make(chan bool, 1)
)

const (
	preferredHorizontalThreshold = 5
	tabstopLength                = 8
)

func tbPrint(x, y int, fg, bg termbox.Attribute, msg string) {
	for _, c := range msg {
		termbox.SetCell(x, y, c, fg, bg)
		x += runewidth.RuneWidth(c)
	}
}

func fill(x, y, w, h int, cell termbox.Cell) {
	for ly := 0; ly < h; ly++ {
		for lx := 0; lx < w; lx++ {
			termbox.SetCell(x+lx, y+ly, cell.Ch, cell.Fg, cell.Bg)
		}
	}
}

func runeAdvanceLen(r rune, pos int) int {
	if r == '\t' {
		return tabstopLength - pos%tabstopLength
	}
	return runewidth.RuneWidth(r)
}

func voffsetCoffset(text []byte, boffset int) (voffset, coffset int) {
	text = text[:boffset]
	for len(text) > 0 {
		r, size := utf8.DecodeRune(text)
		text = text[size:]
		coffset += 1
		voffset += runeAdvanceLen(r, voffset)
	}
	return
}

func byteSliceGrow(s []byte, desiredCap int) []byte {
	if cap(s) < desiredCap {
		ns := make([]byte, len(s), desiredCap)
		copy(ns, s)
		return ns
	}
	return s
}

func byteSliceRemove(text []byte, from, to int) []byte {
	size := to - from
	copy(text[from:], text[to:])
	text = text[:len(text)-size]
	return text
}

func byteSliceInsert(text []byte, offset int, what []byte) []byte {
	n := len(text) + len(what)
	text = byteSliceGrow(text, n)
	text = text[:n]
	copy(text[offset+len(what):], text[offset:])
	copy(text[offset:], what)
	return text
}

type InputBox struct {
	text          []byte
	lineVoffset   int
	cursorBoffset int // cursor offset in bytes
	cursorVoffset int // visual cursor offset in termbox cells
	cursorCoffset int // cursor offset in unicode code points
}

func (ib *InputBox) Clear() {
	ib.text = nil
	ib.lineVoffset = 0
	ib.cursorBoffset = 0
	ib.cursorVoffset = 0
	ib.cursorCoffset = 0
}

func (ib *InputBox) Draw(x, y, w, h int) {
	ib.AdjustVOffset(w)

	fill(x, y, w, h, termbox.Cell{Ch: ' '})

	t := ib.text
	lx := 0
	tabstop := 0
	for {
		rx := lx - ib.lineVoffset
		if len(t) == 0 {
			break
		}

		if lx == tabstop {
			tabstop += tabstopLength
		}

		if rx >= w {
			termbox.SetCell(x+w-1, y, '→',
				colorDefault, colorDefault)
			break
		}

		r, size := utf8.DecodeRune(t)
		if r == '\t' {
			for ; lx < tabstop; lx++ {
				rx = lx - ib.lineVoffset
				if rx >= w {
					goto next
				}

				if rx >= 0 {
					termbox.SetCell(x+rx, y, ' ', colorDefault, colorDefault)
				}
			}
		} else {
			if rx >= 0 {
				termbox.SetCell(x+rx, y, r, colorDefault, colorDefault)
			}
			lx += runewidth.RuneWidth(r)
		}
	next:
		t = t[size:]
	}

	if ib.lineVoffset != 0 {
		termbox.SetCell(x, y, '←', colorDefault, colorDefault)
	}
}

func (ib *InputBox) AdjustVOffset(width int) {
	ht := preferredHorizontalThreshold
	maxHThreshold := (width - 1) / 2
	if ht > maxHThreshold {
		ht = maxHThreshold
	}

	threshold := width - 1
	if ib.lineVoffset != 0 {
		threshold = width - ht
	}
	if ib.cursorVoffset-ib.lineVoffset >= threshold {
		ib.lineVoffset = ib.cursorVoffset + (ht - width + 1)
	}

	if ib.lineVoffset != 0 && ib.cursorVoffset-ib.lineVoffset < ht {
		ib.lineVoffset = ib.cursorVoffset - ht
		if ib.lineVoffset < 0 {
			ib.lineVoffset = 0
		}
	}
}

func (ib *InputBox) MoveCursorTo(boffset int) {
	ib.cursorBoffset = boffset
	ib.cursorVoffset, ib.cursorCoffset = voffsetCoffset(ib.text, boffset)
}

func (ib *InputBox) RuneUnderCursor() (rune, int) {
	return utf8.DecodeRune(ib.text[ib.cursorBoffset:])
}

func (ib *InputBox) RuneBeforeCursor() (rune, int) {
	return utf8.DecodeLastRune(ib.text[:ib.cursorBoffset])
}

func (ib *InputBox) MoveCursorOneRuneBackward() {
	if ib.cursorBoffset == 0 {
		return
	}
	_, size := ib.RuneBeforeCursor()
	ib.MoveCursorTo(ib.cursorBoffset - size)
}

func (ib *InputBox) MoveCursorOneRuneForward() {
	if ib.cursorBoffset == len(ib.text) {
		return
	}
	_, size := ib.RuneUnderCursor()
	ib.MoveCursorTo(ib.cursorBoffset + size)
}

func (ib *InputBox) MoveCursorToBeginningOfTheLine() {
	ib.MoveCursorTo(0)
}

func (ib *InputBox) MoveCursorToEndOfTheLine() {
	ib.MoveCursorTo(len(ib.text))
}

func (ib *InputBox) DeleteRuneBackward() {
	if ib.cursorBoffset == 0 {
		return
	}

	ib.MoveCursorOneRuneBackward()
	_, size := ib.RuneUnderCursor()
	ib.text = byteSliceRemove(ib.text, ib.cursorBoffset, ib.cursorBoffset+size)
}

func (ib *InputBox) DeleteRuneForward() {
	if ib.cursorBoffset == len(ib.text) {
		return
	}
	_, size := ib.RuneUnderCursor()
	ib.text = byteSliceRemove(ib.text, ib.cursorBoffset, ib.cursorBoffset+size)
}

func (ib *InputBox) DeleteTheRestOfTheLine() {
	ib.text = ib.text[:ib.cursorBoffset]
}

func (ib *InputBox) InsertRune(r rune) {
	var buf [utf8.UTFMax]byte
	n := utf8.EncodeRune(buf[:], r)
	ib.text = byteSliceInsert(ib.text, ib.cursorBoffset, buf[:n])
	ib.MoveCursorOneRuneForward()
}

func (ib *InputBox) CursorX() int {
	return ib.cursorVoffset - ib.lineVoffset
}

type MessageBox struct {
	text    [][]byte
	maxLine int
}

func (mb *MessageBox) Draw(x int, y int, w int, h int) {
	records := mb.text
	if len(mb.text) == 0 {
		return
	}
	const coldef = termbox.ColorDefault
	rx := 0
	ry := y

	for _, record := range records {
		rx = 0
		ry += 1
		for len(record) > 0 {
			// 换行
			if rx >= w {
				rx = 0
				ry += 1
			}

			r, size := utf8.DecodeRune(record)
			if r == '\t' {
				for i := 0; i < tabstopLength; i++ {
					rx += 1
					if rx >= w {
						goto next
					}

					if rx >= 0 {
						termbox.SetCell(x+rx, ry, ' ', coldef, coldef)
					}
				}
			} else if r == '\n' {
				rx = 0
				ry += 1
			} else {
				if rx >= 0 {
					termbox.SetCell(x+rx, ry, r, coldef, coldef)
				}
				rx += runewidth.RuneWidth(r)
			}
		next:
			record = record[size:]
		}
	}
}

func (mb *MessageBox) AppendAndRedraw(text []byte) {
	mb.Append(text)
	redrawAll()
}

func (mb *MessageBox) Append(text []byte) {
	mb.text = append(mb.text, text)
	l := len(mb.text)
	if l > mb.maxLine {
		mb.text = mb.text[l-mb.maxLine:]
	}
}

func redrawPrepare() {
	colorDefault = termbox.ColorDefault
	termbox.Clear(colorDefault, colorDefault)

	termW, termH = termbox.Size()

	if termH < 6 {
		log.Errorf("窗口太小了")
		time.Sleep(3 * time.Second)
		Quit()
	}

	termX = 0
	termY = 0
}

func redrawAll() {
	redrawPrepare()

	inputX := termX
	inputY := termH - 2

	fill(inputX, inputY-1, termW, 1, termbox.Cell{Ch: '─'})

	messageBox.maxLine = inputY - 3
	messageBox.Draw(termX, termY, termW, termH)
	inputBox.Draw(inputX, inputY, termW, 1)

	termbox.SetCursor(inputX+inputBox.CursorX(), inputY)

	termbox.Flush()
}

func New() (chan []byte, chan []byte, error) {
	err := termbox.Init()
	if err != nil {
		return nil, nil, err
	}

	termbox.SetInputMode(termbox.InputEsc)
	redrawAll()

	// 输出显示
	go func() {
		for o := range outputChan {
			messageBox.AppendAndRedraw(o)
		}
	}()

	return inputChan, outputChan, nil
}

func StartInput() {
	inputCtlChan <- true
}

func StopInput() {
	inputCtlChan <- false
}

func Quit() {
	if termbox.IsInit {
		termbox.Interrupt()
		termbox.Close()
	}
}

func Start() {

	go func() {
		isInput := false
		for {
			select {
			case ctl := <-inputCtlChan:
				isInput = ctl
				inputBox.Clear()
			case ev := <-eventChan:
				switch ev.Type {
				case termbox.EventKey:
					if isInput {
						switch ev.Key {
						case termbox.KeyEsc, termbox.KeyCtrlC:
							Quit()
							return
						case termbox.KeyArrowLeft, termbox.KeyCtrlB:
							inputBox.MoveCursorOneRuneBackward()
						case termbox.KeyArrowRight, termbox.KeyCtrlF:
							inputBox.MoveCursorOneRuneForward()
						case termbox.KeyBackspace, termbox.KeyBackspace2:
							inputBox.DeleteRuneBackward()
						case termbox.KeyDelete, termbox.KeyCtrlD:
							inputBox.DeleteRuneForward()
						case termbox.KeyTab:
							inputBox.InsertRune('\t')
						case termbox.KeySpace:
							inputBox.InsertRune(' ')
						case termbox.KeyCtrlK:
							inputBox.DeleteTheRestOfTheLine()
						case termbox.KeyHome, termbox.KeyCtrlA:
							inputBox.MoveCursorToBeginningOfTheLine()
						case termbox.KeyEnd, termbox.KeyCtrlE:
							inputBox.MoveCursorToEndOfTheLine()
						case termbox.KeyEnter:
							if len(inputBox.text) > 0 {
								text := inputBox.text
								inputChan <- text
								inputBox.Clear()
							}
							continue
						default:
							if ev.Ch != 0 {
								inputBox.InsertRune(ev.Ch)
							}
						}
					} else {
						switch ev.Key {
						case termbox.KeyEsc, termbox.KeyCtrlC:
							Quit()
							return
						}
					}

				case termbox.EventError:
					//panic(ev.Err)
				}
			}

			redrawAll()
		}
	}()

	for {
		ev := termbox.PollEvent()
		if ev.Type == termbox.EventInterrupt {
			return
		}
		eventChan <- ev
	}
}
