package genes

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/antonybholmes/go-dna"
	dnaroutes "github.com/antonybholmes/go-edb-server-gin/routes/modules/dna"
	"github.com/antonybholmes/go-genome"
	"github.com/antonybholmes/go-genome/genomedbcache"
	basemath "github.com/antonybholmes/go-math"
	"github.com/antonybholmes/go-web"
	"github.com/gin-gonic/gin"
)

const DEFAULT_LEVEL genome.Level = genome.LEVEL_GENE

const DEFAULT_CLOSEST_N uint16 = 5

// A GeneQuery contains info from query params.
type GeneQuery struct {
	Level    genome.Level
	Db       *genome.GeneDB
	Assembly string
	GeneType string // e.g. "protein_coding", "non_coding", etc.
	// only show canonical genes
	Canonical bool
}

type GenesResp struct {
	Location *dna.Location            `json:"location"`
	Features []*genome.GenomicFeature `json:"features"`
}

const MAX_ANNOTATIONS = 1000

type AnnotationResponse struct {
	Status int                      `json:"status"`
	Data   []*genome.GeneAnnotation `json:"data"`
}

func parseGeneQuery(c *gin.Context, assembly string) (*GeneQuery, error) {

	var level genome.Level = DEFAULT_LEVEL

	switch c.Query("level") {
	case "exon":
		level = genome.LEVEL_EXON
	case "transcript":
		level = genome.LEVEL_TRANSCRIPT
	default:
		level = genome.LEVEL_GENE

	}

	canonical := strings.HasPrefix(strings.ToLower(c.Query("canonical")), "t")

	geneType := c.Query("type")

	// user can specify gene type in query string, but we sanitize it
	switch {
	case strings.Contains(geneType, "protein"):
		geneType = "protein_coding"
	default:
		geneType = ""
	}

	db, err := genomedbcache.GeneDB(assembly)

	if err != nil {
		return nil, fmt.Errorf("unable to open database for assembly %s %s", assembly, err)
	}

	return &GeneQuery{
			Assembly:  assembly,
			GeneType:  geneType,
			Db:        db,
			Level:     level,
			Canonical: canonical},
		nil
}

// func GeneDBInfoRoute(c *gin.Context) {
// 	query, err := ParseGeneQuery(c, c.Param("assembly"))

// 	if err != nil {
// 		return web.ErrorReq(err)
// 	}

// 	info, _ := query.Db.GeneDBInfo()

// 	// if err != nil {
// 	// 	return web.ErrorReq(err)
// 	// }

// 	web.MakeDataResp(c, "", &info)
// }

func GenomesRoute(c *gin.Context) {
	infos, err := genomedbcache.GetInstance().List()

	if err != nil {
		c.Error(err)
		return
	}

	web.MakeDataResp(c, "", infos)
}

func OverlappingGenesRoute(c *gin.Context) {
	locations, err := dnaroutes.ParseLocationsFromPost(c) // dnaroutes.ParseLocationsFromPost(c)

	if err != nil {
		c.Error(err)
		return
	}

	query, err := parseGeneQuery(c, c.Param("assembly"))

	if err != nil {
		c.Error(err)
		return
	}

	if len(locations) == 0 {
		web.BadReqResp(c, "must supply at least 1 location")
	}

	ret := make([]*GenesResp, 0, len(locations))

	for _, location := range locations {
		features, err := query.Db.OverlappingGenes(location, query.Canonical, query.GeneType)

		if err != nil {
			c.Error(err)
			return
		}

		ret = append(ret, &GenesResp{Location: location, Features: features})

	}

	web.MakeDataResp(c, "", &ret)
}

func SearchForGeneByNameRoute(c *gin.Context) {
	search := c.Query("search") // dnaroutes.ParseLocationsFromPost(c)

	if search == "" {
		web.BadReqResp(c, "search cannot be empty")
		return
	}

	fuzzyMode := c.Query("mode") == "fuzzy"

	n := web.ParseN(c, 20)

	query, err := parseGeneQuery(c, c.Param("assembly"))

	if err != nil {
		c.Error(err)
		return
	}

	canonical := strings.HasPrefix(strings.ToLower(c.Query("canonical")), "t")

	features, _ := query.Db.SearchForGeneByName(search, query.Level, n, fuzzyMode, canonical, c.Query("type"))

	// if err != nil {
	// 	return web.ErrorReq(err)
	// }

	web.MakeDataResp(c, "", &features)
}

func WithinGenesRoute(c *gin.Context) {
	locations, err := dnaroutes.ParseLocationsFromPost(c) // dnaroutes.ParseLocationsFromPost(c)

	if err != nil {
		c.Error(err)
		return
	}

	query, err := parseGeneQuery(c, c.Param("assembly"))

	if err != nil {
		c.Error(err)
		return
	}

	data := make([]*genome.GenomicFeatures, len(locations))

	for li, location := range locations {
		genes, err := query.Db.WithinGenes(location, query.Level)

		if err != nil {
			c.Error(err)
			return
		}

		data[li] = genes
	}

	web.MakeDataResp(c, "", &data)
}

// Find the n closest genes to a location
func ClosestGeneRoute(c *gin.Context) {
	locations, err := dnaroutes.ParseLocationsFromPost(c)

	if err != nil {
		c.Error(err)
		return
	}

	query, err := parseGeneQuery(c, c.Param("assembly"))

	if err != nil {
		c.Error(err)
		return
	}

	n := web.ParseN(c, DEFAULT_CLOSEST_N)

	data := make([]*genome.GenomicFeatures, len(locations))

	for li, location := range locations {
		genes, err := query.Db.ClosestGenes(location, n, query.Level)

		if err != nil {
			c.Error(err)
			return
		}

		data[li] = genes
	}

	web.MakeDataResp(c, "", &data)
}

func ParseTSSRegion(c *gin.Context) *dna.TSSRegion {

	v := c.Query("tss")

	if v == "" {
		return dna.NewTSSRegion(2000, 1000)
	}

	tokens := strings.Split(v, ",")

	s, err := strconv.ParseUint(tokens[0], 10, 0)

	if err != nil {
		return dna.NewTSSRegion(2000, 1000)
	}

	e, err := strconv.ParseUint(tokens[1], 10, 0)

	if err != nil {
		return dna.NewTSSRegion(2000, 1000)
	}

	return dna.NewTSSRegion(uint(s), uint(e))
}

func AnnotateRoute(c *gin.Context) {
	locations, err := dnaroutes.ParseLocationsFromPost(c)

	if err != nil {
		c.Error(err)
		return
	}

	// limit amount of data returned per request to 1000 entries at a time
	locations = locations[0:basemath.Min(len(locations), MAX_ANNOTATIONS)]

	query, err := parseGeneQuery(c, c.Param("assembly"))

	if err != nil {
		c.Error(err)
		return
	}

	n := web.ParseN(c, DEFAULT_CLOSEST_N)

	tssRegion := ParseTSSRegion(c)

	output := web.ParseOutput(c)

	annotationDb := genome.NewAnnotateDb(query.Db, tssRegion, n)

	data := make([]*genome.GeneAnnotation, len(locations))

	for li, location := range locations {

		annotations, err := annotationDb.Annotate(location)

		if err != nil {
			c.Error(err)
			return
		}

		data[li] = annotations
	}

	if output == "text" {
		tsv, err := MakeGeneTable(data, tssRegion)

		if err != nil {
			c.Error(err)
			return
		}

		c.String(http.StatusOK, tsv)
	} else {

		c.JSON(http.StatusOK, AnnotationResponse{Status: http.StatusOK, Data: data})
	}
}

func MakeGeneTable(
	data []*genome.GeneAnnotation,
	ts *dna.TSSRegion,
) (string, error) {
	var buffer bytes.Buffer
	wtr := csv.NewWriter(&buffer)
	wtr.Comma = '\t'

	closestN := len(data[0].ClosestGenes)

	headers := make([]string, 6+5*closestN)

	headers[0] = "Location"
	headers[1] = "ID"
	headers[2] = "Gene Symbol"
	headers[3] = fmt.Sprintf(
		"Relative To Gene (prom=-%d/+%dkb)",
		ts.Offset5P()/1000,
		ts.Offset3P()/1000)
	headers[4] = "TSS Distance"
	headers[5] = "Gene Location"

	idx := 6
	for i := 1; i <= closestN; i++ {
		headers[idx] = fmt.Sprintf("#%d Closest ID", i)
		headers[idx] = fmt.Sprintf("#%d Closest Gene Symbols", i)
		headers[idx] = fmt.Sprintf(
			"#%d Relative To Closet Gene (prom=-%d/+%dkb)",
			i,
			ts.Offset5P()/1000,
			ts.Offset3P()/1000)
		headers[idx] = fmt.Sprintf("#%d TSS Closest Distance", i)
		headers[idx] = fmt.Sprintf("#%d Gene Location", i)
	}

	err := wtr.Write(headers)

	if err != nil {
		return "", err
	}

	for _, annotation := range data {
		row := []string{annotation.Location.String(),
			annotation.GeneIds,
			annotation.GeneSymbols,
			annotation.PromLabels,
			annotation.TSSDists,
			annotation.Locations}

		for _, closestGene := range annotation.ClosestGenes {
			row = append(row, closestGene.GeneId)
			row = append(row, genome.GeneWithStrandLabel(closestGene.GeneSymbol, closestGene.Strand))
			row = append(row, closestGene.PromLabel)
			row = append(row, strconv.Itoa(closestGene.TssDist))
			row = append(row, closestGene.Location.String())
		}

		err := wtr.Write(row)

		if err != nil {
			return "", err
		}
	}

	wtr.Flush()

	return buffer.String(), nil
}
