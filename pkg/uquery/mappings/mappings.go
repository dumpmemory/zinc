/* Copyright 2022 Zinc Labs Inc. and Contributors
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package mappings

import (
	"fmt"
	"strings"

	"github.com/blugelabs/bluge/analysis"

	"github.com/zinclabs/zinc/pkg/errors"
	"github.com/zinclabs/zinc/pkg/meta"
	zincanalysis "github.com/zinclabs/zinc/pkg/uquery/analysis"
	"github.com/zinclabs/zinc/pkg/zutils"
)

func Request(analyzers map[string]*analysis.Analyzer, data map[string]interface{}) (*meta.Mappings, error) {
	if len(data) == 0 {
		return nil, nil
	}

	if data["properties"] == nil {
		return nil, errors.New(errors.ErrorTypeParsingException, "[mappings] properties should be defined")

	}

	properties, ok := data["properties"].(map[string]interface{})
	if !ok {
		return nil, errors.New(errors.ErrorTypeParsingException, "[mappings] properties should be an object")
	}

	mappings := meta.NewMappings()
	for field, prop := range properties {
		prop, ok := prop.(map[string]interface{})
		if !ok {
			return nil, errors.New(errors.ErrorTypeParsingException, fmt.Sprintf("[mappings] properties [%s] should be an object", field))
		}
		if v, ok := prop["properties"]; ok {
			if _, ok := v.(map[string]interface{}); !ok {
				return nil, errors.New(errors.ErrorTypeParsingException, fmt.Sprintf("[mappings] properties [%s] should be an object", field))
			}
			if subMappings, err := Request(analyzers, prop); err == nil {
				for k, v := range subMappings.ListProperty() {
					mappings.SetProperty(field+"."+k, v)
				}
			} else {
				return nil, err
			}
			continue
		}
		propType, ok := prop["type"]
		if !ok {
			return nil, errors.New(errors.ErrorTypeParsingException, fmt.Sprintf("[mappings] properties [%s] should be exists", "type"))
		}
		propTypeStr, ok := propType.(string)
		if !ok {
			return nil, errors.New(errors.ErrorTypeParsingException, fmt.Sprintf("[mappings] properties [%s] should be an string", "type"))
		}

		var newProp meta.Property
		propTypeStr = strings.ToLower(propTypeStr)
		switch propTypeStr {
		case "text", "keyword", "numeric", "bool", "date":
			newProp = meta.NewProperty(propTypeStr)
		case "constant_keyword":
			newProp = meta.NewProperty("keyword")
		case "match_only_text":
			newProp = meta.NewProperty("text")
		case "integer", "double", "long", "short", "int", "float":
			newProp = meta.NewProperty("numeric")
		case "boolean":
			newProp = meta.NewProperty("bool")
		case "time", "datetime":
			newProp = meta.NewProperty("date")
		case "flattened", "object", "nested", "wildcard", "byte", "alias", "geo_point", "ip", "ip_range", "scaled_float":
			// ignore
		default:
			return nil, errors.New(errors.ErrorTypeXContentParseException, fmt.Sprintf("[mappings] properties [%s] doesn't support type [%s]", field, propTypeStr))
		}

		for k, v := range prop {
			switch k {
			case "type":
				// handled
			case "analyzer":
				newProp.Analyzer = v.(string)
			case "search_analyzer":
				newProp.SearchAnalyzer = v.(string)
			case "format":
				newProp.Format = v.(string)
			case "time_zone":
				newProp.TimeZone = v.(string)
				_, err := zutils.ParseTimeZone(newProp.TimeZone)
				if err != nil {
					return nil, errors.New(errors.ErrorTypeParsingException, fmt.Sprintf("[mappings] %s time_zone parse err %s", field, err.Error()))
				}
			case "index":
				newProp.Index = v.(bool)
			case "store":
				newProp.Store = v.(bool)
			case "sortable":
				newProp.Sortable = v.(bool)
			case "aggregatable":
				newProp.Aggregatable = v.(bool)
			case "highlightable":
				newProp.Highlightable = v.(bool)
			default:
				// ignore unknown options
				// return nil, errors.New(errors.ErrorTypeParsingException, fmt.Sprintf("[mappings] properties [%s] unknown option [%s]", field, k))
			}
		}

		if newProp.Highlightable {
			newProp.Store = true
		}

		if newProp.Type != "" {
			mappings.SetProperty(field, newProp)
		}

		// check analyzer
		if newProp.Type == "text" {
			if newProp.Analyzer != "" {
				if _, err := zincanalysis.QueryAnalyzer(analyzers, newProp.Analyzer); err != nil {
					return nil, err
				}
			}
			if newProp.SearchAnalyzer != "" {
				if _, err := zincanalysis.QueryAnalyzer(analyzers, newProp.SearchAnalyzer); err != nil {
					return nil, err
				}
			}
		}
	}

	return mappings, nil
}
