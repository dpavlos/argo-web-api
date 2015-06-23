/*
 * Copyright (c) 2014 GRNET S.A., SRCE, IN2P3 CNRS Computing Centre
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the
 * License. You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an "AS
 * IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
 * express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 *
 * The views and conclusions contained in the software and
 * documentation are those of the authors and should not be
 * interpreted as representing official policies, either expressed
 * or implied, of either GRNET S.A., SRCE or IN2P3 CNRS Computing
 * Centre
 *
 * The work represented by this source file is partially funded by
 * the EGI-InSPIRE project through the European Commission's 7th
 * Framework Programme (contract # INFSO-RI-261323)
 */

package recomputations

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"code.google.com/p/gcfg"
	"github.com/argoeu/argo-web-api/utils/config"
	"github.com/stretchr/testify/suite"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

// This is a util. suite struct used in tests (see pkg "testify")
type RecomputationsProfileTestSuite struct {
	suite.Suite
	cfg                       config.Config
	tenantdb                  string
	tenantpassword            string
	tenantusername            string
	tenantstorename           string
	clientkey                 string
	respRecomputationsCreated string
	respUnauthorized          string
}

// Setup the Test Environment
// This function runs before any test and setups the environment
func (suite *RecomputationsProfileTestSuite) SetupTest() {

	const testConfig = `
    [server]
    bindip = ""
    port = 8080
    maxprocs = 4
    cache = false
    lrucache = 700000000
    gzip = true
    [mongodb]
    host = "127.0.0.1"
    port = 27017
    db = "AR_test_recomputations"
    `

	_ = gcfg.ReadStringInto(&suite.cfg, testConfig)

	suite.respRecomputationsCreated = " <root>\n" +
		"   <Message>A recalculation request has been filed</Message>\n </root>"

	suite.respUnauthorized = "Unauthorized"
	suite.tenantdb = "AR_test_recomputations_tenant"
	suite.clientkey = "mysecretcombination"

	// seed mongo
	session, err := mgo.Dial(suite.cfg.MongoDB.Host)
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Seed database with tenants
	//TODO: move tests to
	c := session.DB(suite.cfg.MongoDB.Db).C("tenants")
	c.Insert(
		bson.M{"name": "Westeros",
			"db_conf": []bson.M{

				bson.M{
					"server":   "localhost",
					"port":     27017,
					"database": "argo_Westeros1",
				},
				bson.M{
					"server":   "localhost",
					"port":     27017,
					"database": "argo_Westeros2",
				},
			},
			"users": []bson.M{

				bson.M{
					"name":    "John Snow",
					"email":   "J.Snow@brothers.wall",
					"api_key": "wh1t3_w@lk3rs",
				},
				bson.M{
					"name":    "King Joffrey",
					"email":   "g0dk1ng@kingslanding.gov",
					"api_key": "sansa <3",
				},
			}})
	c.Insert(
		bson.M{"name": "EGI",
			"db_conf": []bson.M{

				bson.M{
					// "store":    "ar",
					"server":   "localhost",
					"port":     27017,
					"database": suite.tenantdb,
					"username": suite.tenantusername,
					"password": suite.tenantpassword,
				},
				bson.M{
					"server":   "localhost",
					"port":     27017,
					"database": "argo_egi_metric_data",
				},
			},
			"users": []bson.M{

				bson.M{
					"name":    "Joe Complex",
					"email":   "C.Joe@egi.eu",
					"api_key": suite.clientkey,
				},
				bson.M{
					"name":    "Josh Plain",
					"email":   "P.Josh@egi.eu",
					"api_key": "itsamysterytoyou",
				},
			}})
	// Seed database with recomputations
	c = session.DB(suite.tenantdb).C("recomputations")
	c.Insert(
		MongoInterface{
			StartTime:   "2015-03-10T12:00:00Z",
			EndTime:     "2015-03-30T23:00:00Z",
			Reason:      "reasons",
			Group:       "NGI_PL",
			ExcludeSite: []string{"WCSS"},
			Status:      "pending",
			Timestamp:   "2015-04-01 14:58:40",
		},
	)
	c.Insert(
		MongoInterface{
			StartTime:   "2015-01-10T12:00:00Z",
			EndTime:     "2015-01-30T23:00:00Z",
			Reason:      "power cuts",
			Group:       "NGI_FR",
			ExcludeSite: []string{"Gluster"},
			Status:      "running",
			Timestamp:   "2015-02-01 14:58:40",
		},
	)

}

func (suite *RecomputationsProfileTestSuite) TestListRecomputations() {

	request, _ := http.NewRequest("GET", "/api/v1/recomputations", strings.NewReader(""))
	request.Header.Set("x-api-key", "mysecretcombination")

	code, _, output, _ := List(request, suite.cfg)

	recomputationRequestsXML := `<root>
 <Request start_time="2015-01-10T12:00:00Z" end_time="2015-01-30T23:00:00Z" reason="power cuts" group="NGI_FR" status="running" timestamp="2015-02-01 14:58:40">
  <Exclude site="Gluster"></Exclude>
 </Request>
 <Request start_time="2015-03-10T12:00:00Z" end_time="2015-03-30T23:00:00Z" reason="reasons" group="NGI_PL" status="pending" timestamp="2015-04-01 14:58:40">
  <Exclude site="WCSS"></Exclude>
 </Request>
</root>`

	// Check that we must have a 200 ok code
	suite.Equal(200, code, "Internal Server Error")
	// Compare the expected and actual xml response
	suite.Equal(recomputationRequestsXML, string(output), "Response body mismatch")

}

func (suite *RecomputationsProfileTestSuite) TestCreateRecomputations() {

	// postData := `
	// {
	// 	"start_time":   "2015-04-10T12:00:00Z",
	// 	"end_time":     "2015-04-30T23:00:00Z",
	// 	"reason":       "ups problems",
	// 	"group":   "NGI_GRNET",
	// 	"excluded_sites": ["CREAM"]
	// }
	// `

	request, _ := http.NewRequest("POST", "/api/v1/recomputations", strings.NewReader(""))
	request.Header.Set("x-api-key", suite.clientkey)
	request.Header.Set("Content-Type", "application/json;")

	urlValues := url.Values{}
	urlValues.Add("start_time", "2015-04-10T12:00:00Z")
	urlValues.Add("end_time", "2015-04-30T23:00:00Z")
	urlValues.Add("reason", "ups problems")
	urlValues.Add("group", "NGI_GRNET")
	urlValues["excluded_sites"] = []string{"CREAM"}
	request.Form = urlValues
	code, _, output, _ := Create(request, suite.cfg)

	recomputationRequestSubmittedXML := ` <root>
   <Message>A recalculation request has been filed</Message>
 </root>`

	// Check that we must have a 200 ok code
	suite.Equal(200, code, "Internal Server Error")
	// Compare the expected and actual xml response

	suite.Regexp(recomputationRequestSubmittedXML, string(output), "Response body mismatch")

	request, _ = http.NewRequest("GET", "/api/v1/recomputations", strings.NewReader(""))
	request.Header.Set("x-api-key", suite.clientkey)

	code, _, output, _ = List(request, suite.cfg)

	recomputationRequestsXML := `<root>
 <Request start_time="2015-01-10T12:00:00Z" end_time="2015-01-30T23:00:00Z" reason="power cuts" group="NGI_FR" status="running" timestamp="2015-02-01 14:58:40">
  <Exclude site="Gluster"></Exclude>
 </Request>
 <Request start_time="2015-03-10T12:00:00Z" end_time="2015-03-30T23:00:00Z" reason="reasons" group="NGI_PL" status="pending" timestamp="2015-04-01 14:58:40">
  <Exclude site="WCSS"></Exclude>
 </Request>
 <Request start_time="2015-04-10T12:00:00Z" end_time="2015-04-30T23:00:00Z" reason="ups problems" group="NGI_GRNET" status="pending" timestamp=".*"></Request>
</root>`

	// Check that we must have a 200 ok code
	suite.Equal(200, code, "Internal Server Error")
	// Compare the expected and actual xml response
	suite.Regexp(recomputationRequestsXML, string(output), "Response body mismatch")
}

//TearDownTest to tear down every test
func (suite *RecomputationsProfileTestSuite) TearDownTest() {

	session, err := mgo.Dial(suite.cfg.MongoDB.Host)
	if err != nil {
		panic(err)
	}
	session.DB(suite.tenantdb).DropDatabase()
	session.DB(suite.cfg.MongoDB.Db).DropDatabase()

}

func TestRecompuptationsTestSuite(t *testing.T) {
	suite.Run(t, new(RecomputationsProfileTestSuite))
}