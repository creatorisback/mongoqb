// mongodb provides a standardised approach to build a monogo db query by reading the request.  
// version: 0.1

package qb

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"gopkg.in/mgo.v2/bson"
)

var logicalOperators map[string]string
var comparisonOperators map[string]string

func init() {

	logicalOperators = make(map[string]string, 4)
	logicalOperators["(and)"] 	= "$and"
	logicalOperators["(or)"] 	= "$or"
	logicalOperators["(nor)"] 	= "$nor"
	logicalOperators["(not)"] 	= "$not"

	comparisonOperators = make(map[string]string, 4)
	comparisonOperators["(eg)"] 	= "$eg"
	comparisonOperators["(gt)"] 	= "$gt"
	comparisonOperators["(gte)"] 	= "$gte"
	comparisonOperators["(in)"] 	= "$in"
	comparisonOperators["(lt)"] 	= "lt"
	comparisonOperators["(lte)"] 	= "lte"
	comparisonOperators["(ne)"] 	= "ne"
	comparisonOperators["(nin)"] 	= "$nin"

}

// Options is to perform operations on designated key from QueryParamsFilters
// DBPath maps the key to database path or key in mongodb.
// Allowed Values are optional. It is to restrict user request to allow defined values.
// DoBefore is an optional function which is to perform value specific operations.
// eg: If value is a strings which must be treated as bson.ObjectId, we can do such a operations with the help of DoBefore
type Options struct {
	DBPath        string
	AllowedValues []string
	DoBefore      func(string) (interface{}, error)
}


// BuildQuery ...
func BuildQuery(r *http.Request, allowedkeys map[string]Options) (bson.M, error) {
	query := make(bson.M, 0)
	log.Printf("values: %+v", r.URL.Query())

	filterBy := r.URL.Query().Get("filterBy")
	filters := strings.Split(filterBy, ",")

	log.Print("filters", filters)

	// TODO: make this verfication as optional after implementing configurations.
	for _, filterKey := range filters {
		// Verfiy and allow only valid filter types
		var rgx = regexp.MustCompile(`\((.*?)\)`)
		operator := rgx.FindString(filterKey)
		fmt.Println("query special- ", operator)

		filterKey := strings.Replace(filterKey, operator, "", -1)

		if _, isAllowed := allowedkeys[filterKey]; !isAllowed {
			return query, errors.New(filterKey + "- key is not allowed.")
		}

		println("filter key - ", filterKey)
		filterValue := r.URL.Query().Get(filterKey)
		if filterValue == "" {
			return query, errors.New(filterKey + " - got empty values.")
		}

		log.Print("filter value - ", filterValue)
		options := allowedkeys[filterKey]
		if options.DBPath == "" {
			return query, errors.New("invalid key, unable to find the path.")
		}

		for _, allowedValue := range options.AllowedValues {
			if filterValue != allowedValue {
				return query, errors.New("value for key: " + filterKey + "is not allowed.")
			}
		}

		if lop, ok := logicalOperators[operator]; ok {
			logicalQuery, err := options.logicalQuery(query, lop, filterValue)
			if err != nil {
				return query, err
			}

			log.Println("query", query)
			log.Println("logical query", logicalQuery)

			query = logicalQuery
			continue
		}

		query, err := options.defaultQuery(query, operator, filterValue)
		if err != nil {
			return query, err
		}
	}

	return query, nil
}

func (options Options) logicalQuery(query bson.M, operator, values string) (bson.M, error) {

	elements := strings.Split(values, ",")
	log.Print("elements", elements)
	modvalues := []interface{}{}
	for _, ele := range elements {
		if options.DoBefore != nil {
			modifiedValue, err := options.DoBefore(ele)
			if err != nil {
				return query, err
			}

			modvalues = append(modvalues, modifiedValue)
		}

		if len(modvalues) > 0 {
			query[options.DBPath] = bson.M{operator: modvalues}
		}

		query[options.DBPath] = bson.M{operator: elements}
	}

	return query, nil
}

func (options Options) defaultQuery(query bson.M, operator, value string) (bson.M, error) {

	if options.DoBefore != nil {
		modifiedValue, err := options.DoBefore(value)
		if err != nil {
			return query, err
		}
		query[options.DBPath] = modifiedValue
	}

	query[options.DBPath] = value
	return query, nil
}