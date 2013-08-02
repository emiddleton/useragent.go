package useragent

import (
	"flag"
	"log"
	"fmt"
	"strings"
	"regexp"
	"bitbucket.org/zombiezen/goray/yaml/data"
	"bitbucket.org/zombiezen/goray/yaml/parser"
	"os"
)

var (
	useragent_yml_file = flag.String("useragent_yml", "./useragent.yml", "Useragent data")
	application_types        = map[string]ApplicationType{}
	device_types             = map[string]DeviceType{}
	rendering_engines        = map[string]RenderingEngine{}
	manufacturers            = map[string]Manufacturer{}
	applications             = map[string]Application{}
	browser_types            = map[string]BrowserType{}
	browsers                 = []Browser{}
	browser_unknown          *Browser
	operating_systems        = []OperatingSystem{}
	operating_system_unknown *OperatingSystem
)

type ApplicationType struct {
	Key             string
	Name            string
}

type DeviceType struct {
	Key             string
	Name            string
}

type BrowserType struct {
	Key             string
	Name            string
}

type RenderingEngine struct {
	Key             string
	Name            string
}

type Manufacturer struct {
	Id              uint16
	Key             string
	Name            string
}

type Application struct {
	Id	        uint16
	Key             string
	Name	        string
	aliases         []string
	applicationType	string
	manufacturer    string
}

type Browser struct {
	Id              uint16
	Key             string
	Name            string
	aliases	        []string
	excludeList     []string
	browserType     string
	manufacturer    string
	renderingEngine string
	parent          *Browser
	children        []Browser
	versionRegexp   *regexp.Regexp
}

type OperatingSystem struct {
	Id	        uint16
	Key             string
	Name	        string
	aliases	        []string
	excludeList     []string
	manufacturer    string
	deviceType      string
	parent          *OperatingSystem
	children        []OperatingSystem
}

func (b Browser)Group()(g string){
	if b.parent != nil {
		return b.parent.Name
	} else {
		return b.Name
	}
}

func (b Browser)BrowserType()(bt BrowserType){
	if b.browserType == "" && b.parent != nil{
		return b.parent.BrowserType()
	} else {
		return browser_types[b.browserType]
	}
}

func (b Browser)Manufacturer()(m Manufacturer){
	if b.manufacturer == "" && b.parent != nil {
		return b.parent.Manufacturer()
	} else {
		return manufacturers[b.manufacturer]
	}
}

func (b Browser)RenderingEngine()(re RenderingEngine){
	if b.renderingEngine == "" && b.parent != nil {
		return b.parent.RenderingEngine()
	} else {
		return rendering_engines[b.renderingEngine]
	}
}

func (b Browser)getVersionRegexp()(regex *regexp.Regexp){
	if b.versionRegexp == nil && b.parent != nil {
		return b.parent.getVersionRegexp()
	} else {
		return b.versionRegexp
	}
}

func (b Browser)Version(ua string)(v []string){
	regexp := b.getVersionRegexp()
//	fmt.Printf("%#v\n",regexp)
	if regexp != nil {
//		fmt.Printf("%#v\n",regexp)
		rslt := regexp.FindStringSubmatch(ua)
		if rslt != nil {
			if len(rslt) < 4 {
				rslt = append(rslt,"0")
			}
			return rslt
		}
	}
	return v
}

func matchesList(str string, ms []string)(ok bool){
	for _,m := range ms {
		rslt := strings.Index(strings.ToLower(str),strings.ToLower(m))
		if rslt != -1 {
			return true
		}
	}
	return false
}

func (b Browser)matched(ua string)(r Browser, ok bool){
	if matchesList(ua,b.aliases) {
		if len(b.children) > 0 {
			for _,child := range b.children {
//				fmt.Printf("child match: %#v\n",child.Key)
				matchedChild,childMatched := child.matched(ua)
				if childMatched {
					return matchedChild,true
				}
			}
		}
		if matchesList(ua,b.excludeList) == false {
			return b,true
		}
	}
	return r,false
}

func FindBrowser(ua string)(b Browser) {
	for _,browser := range browsers {
//		fmt.Printf("parent match: %#v\n",browser.Key)
		match,browserMatched := browser.matched(ua)
		if browserMatched {
			return match
		}
	}
	return *browser_unknown
}

func initBrowser(parent *Browser, browser_key string, browser_node *parser.Mapping)(b Browser, err error) {
	var browser_id uint16
	b.Key = browser_key
	b.parent = parent
	for _,browser_pair := range browser_node.Pairs {
		browser_key,_ := browser_pair.Key.Data().(string)
		browser_value := browser_pair.Value
		switch browser_key {
		case "id":
			browser_id = uint16(browser_value.Data().(int64))
		case "name":
			b.Name = browser_value.Data().(string)
		case "aliases","exclude_list":
			seq,_ := browser_value.(*parser.Sequence)
			matches := []string{}
			for _,match := range seq.Nodes {
				if match.Data() != nil {
					matches = append(matches,match.Data().(string))
				}
			}
			if browser_key == "aliases" {
				b.aliases = matches
			} else {
				b.excludeList = matches
			}
		case "browser_type":
			b.browserType     = browser_value.Data().(string)
		case "manufacturer":
			b.manufacturer    = browser_value.Data().(string)
		case "rendering_engine":
			b.renderingEngine = browser_value.Data().(string)
		case "children":
			for _, child_pair := range browser_value.(*parser.Mapping).Pairs {
				child_key            := child_pair.Key.Data().(string)
				child_node           := child_pair.Value.(*parser.Mapping)
				child_browser,_      := initBrowser(&b,child_key,child_node)
				b.children = append(b.children,child_browser)
			}
		case "version_regex":
			b.versionRegexp   = regexp.MustCompile(browser_value.Data().(string))
		case "parent":
			panic("parent is no longer supported, nest children instead")
		}
	}
	b.Id = b.Manufacturer().Id << 8 + browser_id
	return b,nil
}

func (os OperatingSystem)Manufacturer()(m Manufacturer){
	if os.manufacturer == "" && os.parent != nil {
		return os.parent.Manufacturer()
	} else {
		return manufacturers[os.manufacturer]
	}
}

func (os OperatingSystem)DeviceType()(dt DeviceType){
	if os.deviceType == "" && os.parent != nil {
		return os.parent.DeviceType()
	} else {
		return device_types[os.deviceType]
	}
}

func (os OperatingSystem)Group()(g string){
	if os.parent != nil {
		return os.parent.Name
	} else {
		return os.Name
	}
}

func (os OperatingSystem)matched(ua string)(r OperatingSystem, ok bool){
	if matchesList(ua,os.aliases) {
		if len(os.children) > 0 {
			for _,child := range os.children {
//				fmt.Printf("child match: %#v\n",child.Key)
				matchedChild,childMatched := child.matched(ua)
				if childMatched {
					return matchedChild,true
				}
			}
		}
		if matchesList(ua,os.excludeList) == false {
			return os,true
		}
	}
	return r,false
}

func FindOperatingSystem(ua string)(os OperatingSystem) {
	for _,os := range operating_systems {
//		fmt.Printf("parent match: %#v\n",os.Key)
		match,osMatched := os.matched(ua)
		if osMatched {
			return match
		}
	}
	return *operating_system_unknown
}

func initOperatingSystem(parent *OperatingSystem, operating_system_key string, operating_system_node *parser.Mapping)(os OperatingSystem, err error) {
	var operating_system_id uint16
	os.Key = operating_system_key
	os.parent = parent
	for _,operating_system_pair := range operating_system_node.Pairs {
		operating_system_key,_ := operating_system_pair.Key.Data().(string)
		operating_system_value := operating_system_pair.Value
		switch operating_system_key {
		case "id":
			operating_system_id = uint16(operating_system_value.Data().(int64))
		case "name":
			os.Name = operating_system_value.Data().(string)
		case "aliases","exclude_list":
			seq,_ := operating_system_value.(*parser.Sequence)
			matches := []string{}
			for _,match := range seq.Nodes {
				if match.Data() != nil {
					matches = append(matches,match.Data().(string))
				}
			}
			if operating_system_key == "aliases" {
				os.aliases = matches
			} else {
				os.excludeList = matches
			}
		case "manufacturer":
			os.manufacturer    = operating_system_value.Data().(string)
		case "device_type":
			os.deviceType      = operating_system_value.Data().(string)
		case "children":
			for _, child_pair := range operating_system_value.(*parser.Mapping).Pairs {
				child_key  := child_pair.Key.Data().(string)
				child_node := child_pair.Value.(*parser.Mapping)
				child_os,_ := initOperatingSystem(&os,child_key,child_node)
				os.children = append(os.children,child_os)
			}
		default:
			panic("unknown tag in operating system")
		}
	}
	os.Id = os.Manufacturer().Id << 8 + operating_system_id
	return os,nil
}

func init() {
	fileReader,err := os.Open(*useragent_yml_file)
	if err != nil {
		log.Fatalf("os.Open(%q): %s", *useragent_yml_file, err)
	}
	yamlParser := parser.New(fileReader,data.CoreSchema,data.DefaultConstructor,nil)
	doc, err := yamlParser.ParseDocument()
	if err != nil {
		log.Fatal(err)
	}
	m, ok := doc.Content.(*parser.Mapping)
	if !ok {
		panic("Top level Yaml is not a map")
	}
	for _,pair := range m.Pairs {
		key_name, ok := pair.Key.Data().(string)
		if !ok {
			panic("Complex type for a map key?")
		}
		key_node, ok := pair.Value.(*parser.Mapping)
		if !ok {
			panic(fmt.Sprintf("Should be map but was %#v",pair.Value))
		}

		switch key_name {
		case "application_types":
			for _,at_pair := range key_node.Pairs {
				at := ApplicationType{}
				at.Key = at_pair.Key.Data().(string)
				at.Name = at_pair.Value.Data().(string)
				application_types[":"+at.Key] = at
			}
		case "manufacturers":
			for _,m_pairs := range key_node.Pairs {
				m := Manufacturer{}
				m.Key = m_pairs.Key.Data().(string)
				m_map := m_pairs.Value.(*parser.Mapping)
				for _,m_pair := range m_map.Pairs {
					key   := m_pair.Key.Data().(string)
					value := m_pair.Value.Data()
					switch key {
					case "id":
						m.Id = uint16(value.(int64))
					case "name":
						m.Name = value.(string)
					default:
						panic("invalid manufacturer key")
					}
				}
				manufacturers[":"+m.Key] = m
//				fmt.Printf("%#v -> %#v\n",m_key,m)
			}
		case "applications":
			for _,a_pairs := range key_node.Pairs {
				a := Application{}
				a.Key = a_pairs.Key.Data().(string)
				a_map := a_pairs.Value.(*parser.Mapping)
				for _,a_pair := range a_map.Pairs {
					key   := a_pair.Key.Data().(string)
					value := a_pair.Value
					switch key {
					case "id":
						a.Id = uint16(value.Data().(int64))
					case "name":
						a.Name = value.Data().(string)
					case "aliases":
						seq,_ := value.(*parser.Sequence)
						matches := []string{}
						for _,match := range seq.Nodes {
							if match.Data() != nil {
								matches = append(matches,match.Data().(string))
							}
						}
						a.aliases = matches
					case "application_type":
						a.applicationType = value.Data().(string)
					case "manufacturer":
						a.manufacturer = value.Data().(string)
					default:
						panic("invalid device type key")
					}
				}
				applications[":"+a.Key] = a
//				fmt.Printf("%#v\n",a)
			}
		case "device_types":
			for _,dt_pair := range key_node.Pairs {
				dt := DeviceType{}
				dt.Key = dt_pair.Key.Data().(string)
				dt.Name = dt_pair.Value.Data().(string)
				device_types[":"+dt.Key] = dt
//				fmt.Printf("%#v\n",dt)
			}
		case "browser_types":
			for _,bt_pair := range key_node.Pairs {
				bt := BrowserType{}
				bt.Key = bt_pair.Key.Data().(string)
				bt.Name = bt_pair.Value.Data().(string)
				browser_types[":"+bt.Key] = bt
//				fmt.Printf("%#v\n",bt)
			}
		case "rendering_engines":
			for _,re_pair := range key_node.Pairs {
				re := RenderingEngine{}
				re.Key = re_pair.Key.Data().(string)
				re.Name = re_pair.Value.Data().(string)
				rendering_engines[":"+re.Key] = re
//				fmt.Printf("%#v\n",re)
			}
		case "browsers":
			for _,browser_pair := range key_node.Pairs {
				browser_key,_ := browser_pair.Key.Data().(string)
				browser_node,_ := browser_pair.Value.(*parser.Mapping)
//				fmt.Printf("Browser{%#v}\n",browser_key)
				browser,err := initBrowser(nil,browser_key,browser_node)
				if err != nil {
					panic(fmt.Sprintf("error: initOperatingSystem -> %#v",err))
				}
				browsers = append(browsers,browser)
				if browser_key == "unknown" {
					browser_unknown = &browser
				}
//				fmt.Printf("%#v\n",browser)
			}

		case "operating_systems":
			for _,os_pair := range key_node.Pairs {
				os_key,_ := os_pair.Key.Data().(string)
				os_node,_ := os_pair.Value.(*parser.Mapping)
//				fmt.Printf("OperatingSystem{%#v}\n",os_key)
				os,err := initOperatingSystem(nil,os_key,os_node)
				if err != nil {
					panic(fmt.Sprintf("error: initOperatingSystem -> %#v",err))
				}
				operating_systems = append(operating_systems,os)
				if os_key == "unknown" {
					operating_system_unknown = &os
				}
//				fmt.Printf("%#v\n",os)
			}
		}

	}
}
