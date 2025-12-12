package kktemplate

import (
	"fmt"
	html "html/template"
	"os"
	"strings"
	"sync"
	text "text/template"

	"github.com/yetiz-org/goth-kklogger"
	"github.com/yetiz-org/goth-kktranslation"
)

var TemplateRootPath = "./resources/template"
var StructTemplateFrames = []string{"_main", "_header_content", "_header_claim", "_footer_content", "_footer_claim"}
var FuncMap = html.FuncMap{}
var ErrTemplateNotFound = fmt.Errorf("template file not found")

var htmlTemplateMap, frameHtmlTemplateMap = map[string]*html.Template{}, map[string]*html.Template{}
var textTemplateMap = map[string]*text.Template{}
var htmlLocker, textLocker, frameLocker = sync.Mutex{}, sync.Mutex{}, sync.Mutex{}
var frameExist = false

type Engine struct {
	templateRootPath     string
	structTemplateFrames []string
	funcMap              html.FuncMap

	htmlTemplateMap      *map[string]*html.Template
	frameHtmlTemplateMap *map[string]*html.Template
	textTemplateMap      *map[string]*text.Template

	htmlLocker  *sync.Mutex
	textLocker  *sync.Mutex
	frameLocker *sync.Mutex

	frameExist *bool

	getTemplateRootPath     func() string
	setTemplateRootPath     func(string)
	getStructTemplateFrames func() []string
	setStructTemplateFrames func([]string)
	getFuncMap              func() html.FuncMap
	setFuncMap              func(html.FuncMap)
}

var defaultEngine = newDefaultEngine()

func newDefaultEngine() *Engine {
	return &Engine{
		htmlTemplateMap:      &htmlTemplateMap,
		frameHtmlTemplateMap: &frameHtmlTemplateMap,
		textTemplateMap:      &textTemplateMap,
		htmlLocker:           &htmlLocker,
		textLocker:           &textLocker,
		frameLocker:          &frameLocker,
		frameExist:           &frameExist,
		getTemplateRootPath: func() string {
			return TemplateRootPath
		},
		setTemplateRootPath: func(path string) {
			TemplateRootPath = path
		},
		getStructTemplateFrames: func() []string {
			return StructTemplateFrames
		},
		setStructTemplateFrames: func(frames []string) {
			StructTemplateFrames = frames
		},
		getFuncMap: func() html.FuncMap {
			return FuncMap
		},
		setFuncMap: func(fm html.FuncMap) {
			FuncMap = fm
		},
	}
}

func Default() *Engine {
	return defaultEngine
}

func New() *Engine {
	htmlMap := map[string]*html.Template{}
	frameHTMLMap := map[string]*html.Template{}
	textMap := map[string]*text.Template{}
	htmlMu := &sync.Mutex{}
	textMu := &sync.Mutex{}
	frameMu := &sync.Mutex{}
	frameExists := false
	return &Engine{
		templateRootPath:     "./resources/template",
		structTemplateFrames: []string{"_main", "_header_content", "_header_claim", "_footer_content", "_footer_claim"},
		funcMap:              html.FuncMap{},
		htmlTemplateMap:      &htmlMap,
		frameHtmlTemplateMap: &frameHTMLMap,
		textTemplateMap:      &textMap,
		htmlLocker:           htmlMu,
		textLocker:           textMu,
		frameLocker:          frameMu,
		frameExist:           &frameExists,
	}
}

func (e *Engine) SetTemplateRootPath(path string) {
	if e == nil {
		return
	}
	if e.setTemplateRootPath != nil {
		e.setTemplateRootPath(path)
		return
	}
	e.templateRootPath = path
}

func (e *Engine) SetStructTemplateFrames(frames []string) {
	if e == nil {
		return
	}
	if e.setStructTemplateFrames != nil {
		e.setStructTemplateFrames(frames)
		return
	}
	e.structTemplateFrames = frames
}

func (e *Engine) SetFuncMap(fm html.FuncMap) {
	if e == nil {
		return
	}
	if e.setFuncMap != nil {
		e.setFuncMap(fm)
		return
	}
	e.funcMap = fm
}

func (e *Engine) templateRootPathValue() string {
	if e == nil {
		return ""
	}
	if e.getTemplateRootPath != nil {
		return e.getTemplateRootPath()
	}
	return e.templateRootPath
}

func (e *Engine) structTemplateFramesValue() []string {
	if e == nil {
		return nil
	}
	if e.getStructTemplateFrames != nil {
		return e.getStructTemplateFrames()
	}
	return e.structTemplateFrames
}

func (e *Engine) funcMapValue() html.FuncMap {
	if e == nil {
		return nil
	}
	if e.getFuncMap != nil {
		return e.getFuncMap()
	}
	return e.funcMap
}

func LoadHtml(name string, lang string) (*html.Template, error) {
	return defaultEngine.LoadHtml(name, lang)
}

func (e *Engine) LoadHtml(name string, lang string) (*html.Template, error) {
	if e == nil || e.htmlTemplateMap == nil || e.htmlLocker == nil {
		return nil, fmt.Errorf("invalid engine")
	}
	mapName := name + "-" + lang
	if e.isDebug() {
		e.htmlLocker.Lock()
		delete(*e.htmlTemplateMap, mapName)
		e.htmlLocker.Unlock()
	}

	e.htmlLocker.Lock()
	tmpl := (*e.htmlTemplateMap)[mapName]
	e.htmlLocker.Unlock()
	if tmpl != nil {
		return tmpl, nil
	}

	data := func() []byte {
		if data, err := os.ReadFile(e.getRealTemplatePath(name, lang)); !os.IsNotExist(err) {
			return data
		}
		return nil
	}()
	if data == nil {
		return nil, ErrTemplateNotFound
	}

	parsed, err := html.New(mapName).Funcs(e.generateHTMLFuncMap(lang)).Parse(string(data))
	if err != nil {
		return nil, err
	}

	e.htmlLocker.Lock()
	if existing := (*e.htmlTemplateMap)[mapName]; existing != nil {
		e.htmlLocker.Unlock()
		return existing, nil
	}
	(*e.htmlTemplateMap)[mapName] = parsed
	e.htmlLocker.Unlock()
	return parsed, nil
}

func LoadFrameHtml(name string, lang string) (*html.Template, error) {
	return defaultEngine.LoadFrameHtml(name, lang)
}

func (e *Engine) LoadFrameHtml(name string, lang string) (*html.Template, error) {
	if e == nil || e.frameHtmlTemplateMap == nil || e.htmlLocker == nil {
		return nil, fmt.Errorf("invalid engine")
	}
	mapName := name + "-" + lang
	if e.isDebug() {
		e.htmlLocker.Lock()
		delete(*e.frameHtmlTemplateMap, mapName)
		e.htmlLocker.Unlock()
	}

	e.htmlLocker.Lock()
	tmpl := (*e.frameHtmlTemplateMap)[mapName]
	e.htmlLocker.Unlock()
	if tmpl != nil {
		return tmpl, nil
	}

	if !e.frameExistValidate() {
		return nil, ErrTemplateNotFound
	}

	tmplPath := e.getRealTemplatePath(name, lang)
	if tmplPath == "" {
		return nil, ErrTemplateNotFound
	}

	if _, err := os.ReadFile(tmplPath); os.IsNotExist(err) {
		return nil, ErrTemplateNotFound
	}

	filePaths := make([]string, 0, 1+len(e.structTemplateFramesValue()))
	filePaths = append(filePaths, tmplPath)
	for _, structFrame := range e.structTemplateFramesValue() {
		filePaths = append(filePaths, e.getRealTemplatePath(structFrame, lang))
	}

	parsed, err := html.New(tmplPath).Funcs(e.generateHTMLFuncMap(lang)).ParseFiles(filePaths...)
	if err != nil {
		return nil, err
	}

	e.htmlLocker.Lock()
	if existing := (*e.frameHtmlTemplateMap)[mapName]; existing != nil {
		e.htmlLocker.Unlock()
		return existing, nil
	}
	(*e.frameHtmlTemplateMap)[mapName] = parsed
	e.htmlLocker.Unlock()
	return parsed, nil
}

func (e *Engine) getRealTemplatePath(name string, lang string) string {
	tmplPath := fmt.Sprintf("%s/%s/%s.tmpl", e.templateRootPathValue(), lang, name)
	if _, err := os.Stat(tmplPath); !os.IsNotExist(err) {
		return tmplPath
	}

	ml := func() string {
		if slang := strings.Split(lang, "-"); len(slang) > 1 {
			return slang[0]
		}
		return ""
	}()

	tmplPath = fmt.Sprintf("%s/%s/%s.tmpl", e.templateRootPathValue(), ml, name)
	if _, err := os.Stat(tmplPath); !os.IsNotExist(err) {
		return tmplPath
	}

	tmplPath = fmt.Sprintf("%s/default/%s.tmpl", e.templateRootPathValue(), name)
	if _, err := os.Stat(tmplPath); !os.IsNotExist(err) {
		return tmplPath
	}

	return ""
}

func (e *Engine) frameExistValidate() bool {
	if e == nil || e.frameExist == nil || e.frameLocker == nil {
		return false
	}
	e.frameLocker.Lock()
	defer e.frameLocker.Unlock()
	if *e.frameExist {
		return true
	}
	for _, frame := range e.structTemplateFramesValue() {
		framePath := e.getRealTemplatePath(frame, "")
		if framePath == "" {
			kklogger.ErrorJ("kktemplate:_FrameExistValidate", fmt.Sprintf("frame file %s/%s.tmpl is not exist", e.templateRootPathValue(), frame))
			return false
		}

		if _, err := os.Stat(framePath); os.IsNotExist(err) {
			kklogger.ErrorJ("kktemplate:_FrameExistValidate", fmt.Sprintf("frame file %s/%s.tmpl is not exist", e.templateRootPathValue(), frame))
			return false
		}
	}
	*e.frameExist = true
	return true
}

func LoadText(name string, lang string) (*text.Template, error) {
	return defaultEngine.LoadText(name, lang)
}

func (e *Engine) LoadText(name string, lang string) (*text.Template, error) {
	if e == nil || e.textTemplateMap == nil || e.textLocker == nil {
		return nil, fmt.Errorf("invalid engine")
	}
	mapName := name + "-" + lang
	if e.isDebug() {
		e.textLocker.Lock()
		delete(*e.textTemplateMap, mapName)
		e.textLocker.Unlock()
	}

	e.textLocker.Lock()
	tmpl := (*e.textTemplateMap)[mapName]
	e.textLocker.Unlock()
	if tmpl != nil {
		return tmpl, nil
	}

	data := func() []byte {
		if data, err := os.ReadFile(e.getRealTemplatePath(name, lang)); !os.IsNotExist(err) {
			return data
		}
		return nil
	}()
	if data == nil {
		return nil, ErrTemplateNotFound
	}

	parsed, err := text.New(mapName).Funcs(e.generateTEXTFuncMap(lang)).Parse(string(data))
	if err != nil {
		return nil, err
	}

	e.textLocker.Lock()
	if existing := (*e.textTemplateMap)[mapName]; existing != nil {
		e.textLocker.Unlock()
		return existing, nil
	}
	(*e.textTemplateMap)[mapName] = parsed
	e.textLocker.Unlock()
	return parsed, nil
}

func _IsDebug() bool {
	v := os.Getenv("APP_DEBUG")
	if v == "" {
		v = os.Getenv("KKAPP_DEBUG")
	}
	return strings.ToUpper(v) == "TRUE"
}

func (e *Engine) isDebug() bool {
	return _IsDebug()
}

func (e *Engine) generateHTMLFuncMap(lang string) html.FuncMap {
	funcMap := html.FuncMap{
		"T": func(str string) string { return kktranslation.GetLangFile(lang).T(str) },
	}

	for k, v := range e.funcMapValue() {
		funcMap[k] = v
	}

	return funcMap
}

func (e *Engine) generateTEXTFuncMap(lang string) text.FuncMap {
	funcMap := text.FuncMap{
		"T": func(str string) string { return kktranslation.GetLangFile(lang).T(str) },
	}

	for k, v := range e.funcMapValue() {
		funcMap[k] = v
	}

	return funcMap
}
