package geneconvroutes

import (
	"strings"

	"github.com/antonybholmes/go-edb-server-gin/routes"
	geneconv "github.com/antonybholmes/go-geneconv"
	geneconvdbcache "github.com/antonybholmes/go-geneconv/geneconvdbcache"
	"github.com/gin-gonic/gin"
)

type ReqParams struct {
	Searches []string `json:"searches"`
	Exact    bool     `json:"exact"`
}

func ParseParamsFromPost(c *gin.Context) (*ReqParams, error) {

	var params ReqParams

	err := c.Bind(&params)

	if err != nil {
		return nil, err
	}

	return &params, nil
}

// If species is the empty string, species will be determined
// from the url parameters
// func GeneInfoRoute(c *gin.Context, species string) error {
// 	if species == "" {
// 		species = c.Param("species")
// 	}

// 	params, err := ParseParamsFromPost(c)

// 	if err != nil {
// 		return routes.ErrorReq(err)
// 	}

// 	ret := make([]geneconv.Conversion, len(params.Searches))

// 	for ni, search := range params.Searches {

// 		genes, _ := geneconvdb.GeneInfo(search, species, params.Exact)

// 		ret[ni] = geneconv.Conversion{Search: search, Genes: genes}
// 	}

// 	routes.MakeDataResp(c, "", ret)
// }

func ConvertRoute(c *gin.Context) {
	fromSpecies := c.Param("from")
	toSpecies := c.Param("to")

	// if there is no conversion, just use the regular gene info
	// if fromSpecies == toSpecies {
	// 	return GeneInfoRoute(c, fromSpecies)
	// }

	params, err := ParseParamsFromPost(c)

	if err != nil {
		return err
	}

	var ret geneconv.ConversionResults

	fromSpecies = strings.ToLower(fromSpecies)
	//toSpecies = strings.ToLower(toSpecies)

	if fromSpecies == geneconv.HUMAN_SPECIES {
		ret.From = geneconv.HUMAN_TAX
		ret.To = geneconv.MOUSE_TAX
	} else {
		ret.From = geneconv.MOUSE_TAX
		ret.To = geneconv.HUMAN_TAX
	}

	//ret.Conversions = make([]geneconv.Conversion, len(params.Searches))

	for _, search := range params.Searches {

		// Don't care about the errors, just plug empty list into failures
		conversion, _ := geneconvdbcache.Convert(search, fromSpecies, toSpecies, params.Exact)

		ret.Conversions = append(ret.Conversions, conversion)
	}

	routes.MakeDataResp(c, "", ret)

	//routes.MakeDataResp(c, "", mutationdbcache.GetInstance().List())
}
