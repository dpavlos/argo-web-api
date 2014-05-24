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
	"encoding/xml"
	"github.com/argoeu/ar-web-api/utils/authentication"
	"github.com/argoeu/ar-web-api/utils/config"
	"github.com/argoeu/ar-web-api/utils/mongo"
	"labix.org/v2/mgo/bson"
	"net/http"
	"time"
)

func List(w http.ResponseWriter, r *http.Request, cfg config.Config) []byte {

	results := []ApiRecomputationOutput{}

	session := mongo.OpenSession(cfg)

	err := mongo.Find(session, "AR", "recalculations", nil, "timestamp", &results)

	answer, err := xml.MarshalIndent(results, "", " ")

	if err != nil {
		panic(err)
	}

	mongo.CloseSession(session)

	return []byte("<root>" + string(answer) + "</root>")
}

func Create(w http.ResponseWriter, r *http.Request, cfg config.Config) []byte {
	answer := ""
	//only authenticated requests triger the handling code
	if authentication.Authenticate(r.Header, cfg) {
		err := r.ParseForm()
		if err != nil {
			panic(err)
		}
		urlValues := r.Form
		now := time.Now()
		input := ApiRecomputationInput{
			urlValues.Get("start_time"),
			urlValues.Get("end_time"),
			urlValues.Get("reason"),
			urlValues.Get("vo_name"),
			urlValues.Get("ngi_name"),
			urlValues["exclude_site"],
			"pending",
			now,
			//urlValues["exclude_sf"],
			//urlValues["exclude_end_point"],
		}
		query := bson.M{
			"start_time":   input.Start_time,
			"end_time":     input.End_time,
			"reason":       input.Reason,
			"vo":           input.Vo_name,
			"ngi":          input.Ngi_name,
			"status":       input.Status,
			"timestamp":    input.Timestamp,
			"exclude_site": input.Exclude_site,
		}
		session := mongo.OpenSession(cfg)
		err = mongo.Insert(session, "AR", "recalculations", query)
		if err != nil {
			return []byte("ERROR") //TODO
		}
		answer = "A recalculation request has been filed" //Provide the webUI with an appropriate xml/json response
		mongo.CloseSession(session)
	} else {
		answer = http.StatusText(403)
	}
	return []byte(answer)
}
