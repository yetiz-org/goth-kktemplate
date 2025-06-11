package kktemplate

import (
	"fmt"
	html "html/template"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	text "text/template"

	"github.com/yetiz-org/goth-kklogger"
	"github.com/yetiz-org/goth-kktranslation"
)

var TemplateRootPath = "./resources/template"
var htmlTemplateMap, frameHtmlTemplateMap = map[string]*html.Template{}, map[string]*html.Template{}
var textTemplateMap = map[string]*text.Template{}
var htmlLocker, textLocker = sync.Mutex{}, sync.Mutex{}
var StructTemplateFrames = []string{"_main", "_header_content", "_header_claim", "_footer_content", "_footer_claim"}
var frameExist = false
var FuncMap = html.FuncMap{}
var ErrTemplateNotFound = fmt.Errorf("template file not found")

func LoadHtml(name string, lang string) (*html.Template, error) {
	mapName := name + "-" + lang
	if _IsDebug() {
		htmlLocker.Lock()
		delete(htmlTemplateMap, mapName)
		htmlLocker.Unlock()
	}

	if _, f := htmlTemplateMap[mapName]; !f {
		defer htmlLocker.Unlock()
		htmlLocker.Lock()
		if _, f := htmlTemplateMap[mapName]; !f {
			data := func() []byte {
				if data, err := ioutil.ReadFile(getRealTemplatePath(name, lang)); !os.IsNotExist(err) {
					return data
				}

				return nil
			}()

			if data != nil {
				if tmpl, e := html.New(mapName).Funcs(generateHTMLFuncMap(lang)).Parse(string(data)); e == nil {
					htmlTemplateMap[mapName] = tmpl
				} else {
					return nil, e
				}
			} else {
				return nil, ErrTemplateNotFound
			}
		}
	}

	tmpl, _ := htmlTemplateMap[mapName]
	return tmpl, nil
}

func LoadFrameHtml(name string, lang string) (*html.Template, error) {
	mapName := name + "-" + lang
	if _IsDebug() {
		htmlLocker.Lock()
		delete(frameHtmlTemplateMap, mapName)
		htmlLocker.Unlock()
	}

	if _, f := frameHtmlTemplateMap[mapName]; !f {
		if !_FrameExistValidate() {
			return nil, ErrTemplateNotFound
		}

		defer htmlLocker.Unlock()
		htmlLocker.Lock()
		if _, f := frameHtmlTemplateMap[mapName]; !f {
			tmplPath := getRealTemplatePath(name, lang)
			if tmplPath == "" {
				return nil, ErrTemplateNotFound
			}

			if _, err := ioutil.ReadFile(tmplPath); !os.IsNotExist(err) {
				var filePaths []string
				filePaths = append(filePaths, tmplPath)
				for _, structFrame := range StructTemplateFrames {
					filePaths = append(filePaths, getRealTemplatePath(structFrame, lang))
				}

				if tmpl, e := html.New(tmplPath).Funcs(generateHTMLFuncMap(lang)).ParseFiles(filePaths...); e == nil {
					frameHtmlTemplateMap[mapName] = tmpl
				} else {
					return nil, e
				}
			} else {
				return nil, ErrTemplateNotFound
			}
		}
	}

	tmpl, _ := frameHtmlTemplateMap[mapName]
	return tmpl, nil
}

func getRealTemplatePath(name string, lang string) string {
	tmplPath := fmt.Sprintf("%s/%s/%s.tmpl", TemplateRootPath, lang, name)
	if _, err := ioutil.ReadFile(tmplPath); !os.IsNotExist(err) {
		return tmplPath
	}

	ml := func() string {
		if slang := strings.Split(lang, "-"); len(slang) > 1 {
			return slang[0]
		}

		return ""
	}()

	tmplPath = fmt.Sprintf("%s/%s/%s.tmpl", TemplateRootPath, ml, name)
	if _, err := ioutil.ReadFile(tmplPath); !os.IsNotExist(err) {
		return tmplPath
	}

	tmplPath = fmt.Sprintf("%s/default/%s.tmpl", TemplateRootPath, name)
	if _, err := ioutil.ReadFile(tmplPath); !os.IsNotExist(err) {
		return tmplPath
	}

	return ""
}

func _FrameExistValidate() bool {
	if !frameExist {
		for _, frame := range StructTemplateFrames {
			framePath := getRealTemplatePath(frame, "")
			if framePath == "" {
				kklogger.ErrorJ("kktemplate:_FrameExistValidate", fmt.Sprintf("frame file %s/%s.tmpl is not exist", TemplateRootPath, frame))
				return false
			}

			if _, err := ioutil.ReadFile(framePath); os.IsNotExist(err) {
				kklogger.ErrorJ("kktemplate:_FrameExistValidate", fmt.Sprintf("frame file %s/%s.tmpl is not exist", TemplateRootPath, frame))
				return false
			}
		}

		frameExist = true
	}

	return true
}

func LoadText(name string, lang string) (*text.Template, error) {
	mapName := name + "-" + lang
	if _IsDebug() {
		textLocker.Lock()
		delete(textTemplateMap, mapName)
		textLocker.Unlock()
	}

	if _, f := textTemplateMap[mapName]; !f {
		textLocker.Lock()
		defer textLocker.Unlock()

		if _, f := textTemplateMap[mapName]; !f {
			data := func() []byte {
				if data, err := ioutil.ReadFile(getRealTemplatePath(name, lang)); !os.IsNotExist(err) {
					return data
				}

				return nil
			}()

			if data != nil {
				if tmpl, e := text.New(mapName).Funcs(generateTEXTFuncMap(lang)).Parse(string(data)); e == nil {
					textTemplateMap[mapName] = tmpl
				} else {
					return nil, e
				}
			} else {
				return nil, ErrTemplateNotFound
			}
		}
	}

	tmpl, _ := textTemplateMap[mapName]
	return tmpl, nil
}

func _IsDebug() bool {
	return strings.ToUpper(os.Getenv("KKAPP_DEBUG")) == "TRUE"
}

func generateHTMLFuncMap(lang string) html.FuncMap {
	funcMap := html.FuncMap{
		"T": func(str string) string { return kktranslation.GetLangFile(lang).T(str) },
	}

	for k, v := range FuncMap {
		funcMap[k] = v
	}

	return funcMap
}

func generateTEXTFuncMap(lang string) text.FuncMap {
	funcMap := text.FuncMap{
		"T": func(str string) string { return kktranslation.GetLangFile(lang).T(str) },
	}

	for k, v := range FuncMap {
		funcMap[k] = v
	}

	return funcMap
}
