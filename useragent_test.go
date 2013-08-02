package useragent

import (
	"fmt"
	"testing"
	"bitbucket.org/zombiezen/goray/yaml/data"
	"bitbucket.org/zombiezen/goray/yaml/parser"
	"log"
	"os"
)

var (
	browser_parse                = []BrowserParse{}
	browser_version              = []BrowserVersion{}
	browser_incomplete           = []BrowserIncomplete{}
	operating_system_device_type = []OperatingSystemDeviceType{}
	operating_system_parse       = []OperatingSystemParse{}
)

type BrowserParse struct {
	Key        string
	UserAgents []string
}

type Version struct {
	Full  string
	Major string
	Minor string
}

type BrowserVersion struct {
	UserAgent  string
	Version    *Version
}

type BrowserIncomplete string

type OperatingSystemDeviceType struct {
	DeviceType string
	UserAgents []string
}

type OperatingSystemParse struct {
	Key        string
	UserAgents []string
}

func init() {
	fileReader,err := os.Open("useragent_test.yml")
	if err != nil {
		log.Fatalf("os.Open(%q): %s", "useragent_test.yml", err)
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
		case "browser":
			for _,section_pairs := range key_node.Pairs {
				section_key := section_pairs.Key.Data().(string)
				section_value := section_pairs.Value
				switch section_key {
				case "parse":
					for _,test_pairs := range section_value.(*parser.Mapping).Pairs {
						bp := BrowserParse{}
						bp.Key = test_pairs.Key.Data().(string)
						uas := []string{}
						seq,_ := test_pairs.Value.(*parser.Sequence)
						for _,ua := range seq.Nodes {
							uas = append(uas,ua.Data().(string))
						}
						bp.UserAgents = uas
						browser_parse = append(browser_parse,bp)
					}
				case "version":
					seq,_ := section_value.(*parser.Sequence)
					for _,pv := range seq.Nodes {
						bv := BrowserVersion{}
						for _,key_pairs := range pv.(*parser.Mapping).Pairs {
							switch key_pairs.Key.Data().(string) {
							case "user_agent":
								bv.UserAgent = key_pairs.Value.Data().(string)
							case "version":
								v := Version{}
								for _,v_pair := range key_pairs.Value.(*parser.Mapping).Pairs {
									switch v_pair.Key.Data().(string) {
									case "full":
										// fmt.Printf("%#v\n",v_pair.Value.(*parser.Scalar).String())
										v.Full = v_pair.Value.(*parser.Scalar).String()
									case "major":
										// fmt.Printf("%#v\n",v_pair.Value.(*parser.Scalar).String())
										v.Major = v_pair.Value.(*parser.Scalar).String()
									case "minor":
										// fmt.Printf("%#v\n",v_pair.Value.(*parser.Scalar).String())
										v.Minor = v_pair.Value.(*parser.Scalar).String()
									}
								}
								bv.Version = &v
							}
						}
						browser_version = append(browser_version,bv)
					}
				case "incomplete":
				}
			}
		case "operating_system":
			for _,section_pairs := range key_node.Pairs {
				section_key := section_pairs.Key.Data().(string)
				section_value := section_pairs.Value
				switch section_key {
				case "device_type":
					for _,dt_pair := range section_value.(*parser.Mapping).Pairs {
						dt := OperatingSystemDeviceType{}
						dt.DeviceType = dt_pair.Key.Data().(string)
						uas := []string{}
						seq,_ := dt_pair.Value.(*parser.Sequence)
						for _,ua := range seq.Nodes {
							uas = append(uas,ua.Data().(string))
						}
						dt.UserAgents = uas
						operating_system_device_type = append(operating_system_device_type,dt)
					}
				case "parse":
					for _,test_pairs := range section_value.(*parser.Mapping).Pairs {
						osp := OperatingSystemParse{}
						osp.Key = test_pairs.Key.Data().(string)
						uas := []string{}
						seq,_ := test_pairs.Value.(*parser.Sequence)
						for _,ua := range seq.Nodes {
							uas = append(uas,ua.Data().(string))
						}
						osp.UserAgents = uas
						operating_system_parse = append(operating_system_parse,osp)
					}
				}
			}
		}
	}
}

func TestBrowserIdentification(t *testing.T) {
	for _,bp := range browser_parse {
		for _,ua := range bp.UserAgents {
			b := FindBrowser(ua)
			if bp.Key != b.Key {
				t.Errorf("expected: %#v\ngot: %#v\n",bp.Key,b.Key)
			}
			// fmt.Printf("%#v\n",b.Key)
		}
	}
}

func TestBrowserVersionParsing(t *testing.T) {
	for _,bv := range browser_version {
		b := FindBrowser(bv.UserAgent)
		// fmt.Printf("%#v\n",b.Name)
		v := b.Version(bv.UserAgent)
		// fmt.Printf("%#v\n",v)
		if bv.Version == nil {
			if v != nil {
				t.Errorf("Version\nexpected: %#v\ngot: %#v\n",bv.Version,v)
			}
		} else {
			if v[1] != bv.Version.Full {
				t.Errorf("Version.Full\nexpected: %#v\ngot: %#v\n",bv.Version.Full,v[1])
			}
			if v[2] != bv.Version.Major {
				t.Errorf("Version.Major\nexpected: %#v\ngot: %#v\n",bv.Version.Major,v[2])
			}
			if v[3] != bv.Version.Minor {
				t.Errorf("Version.Minor\nexpected: %#v\ngot: %#v\n",bv.Version.Minor,v[3])
			}
		}
	}
}

func TestOperatingSystemDeviceTypeIdentification(t *testing.T) {
	for _,osdt := range operating_system_device_type {
		for _,ua := range osdt.UserAgents {
			os := FindOperatingSystem(ua)
			if osdt.DeviceType != os.DeviceType().Key {
				t.Errorf("OperatingSystem.DeviceType[%#v]\nexpected: %#v\ngot: %#v\n",osdt.DeviceType,osdt.DeviceType,os.DeviceType().Key)
			}
		}
	}
}

func TestOperatingSystemIdentification(t *testing.T) {
	for _,osp := range operating_system_parse {
		for _,ua := range osp.UserAgents {
			os := FindOperatingSystem(ua)
			if osp.Key != os.Key {
				t.Errorf("OperatingSystem\nexpected: %#v\ngot: %#v\n",osp.Key,os.Key)
			}
		}
	}
}
