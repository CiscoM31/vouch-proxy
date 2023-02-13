/*

Copyright 2020 The Vouch Proxy Authors.
Use of this source code is governed by The MIT License (MIT) that
can be found in the LICENSE file. Software distributed under The
MIT License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES
OR CONDITIONS OF ANY KIND, either express or implied.

*/

package openid

import (
	"encoding/json"
	"github.com/vouch/vouch-proxy/pkg/cfg"
	"github.com/vouch/vouch-proxy/pkg/providers/common"
	"github.com/vouch/vouch-proxy/pkg/structs"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"io/ioutil"
	"net/http"
)

// Provider provider specific functions
type Provider struct{}

var log *zap.SugaredLogger

// Configure see main.go configure()
func (Provider) Configure() {
	log = cfg.Logging.Logger
}

// GetUserInfo provider specific call to get userinfomation
func (Provider) GetUserInfo(r *http.Request, user *structs.User, customClaims *structs.CustomClaims, ptokens *structs.PTokens, opts ...oauth2.AuthCodeOption) (rerr error) {
	client, _, err := common.PrepareTokensAndClient(r, ptokens, true, opts...)
	if err != nil {
		return err
	}
	userinfo, err := client.Get(cfg.GenOAuth.UserInfoURL)

	if err != nil {
		return err
	}
	defer func() {
		if err := userinfo.Body.Close(); err != nil {
			rerr = err
		}
	}()
	data, _ := ioutil.ReadAll(userinfo.Body)
	log.Infof("OpenID userinfo body: %s", string(data))
	if err = common.MapClaims(data, customClaims); err != nil {
		log.Error(err)
		return err
	}
	if err = json.Unmarshal(data, user); err != nil {
		log.Error(err)
		return err
	}
	var f interface{}
	err = json.Unmarshal(data, &f)
	if err != nil {
		log.Error(err)
		return err
	}
	if cfg.Cfg.TeamWhiteListClaim != "" {
		m := f.(map[string]interface{})
		for k := range m {
			if k == cfg.Cfg.TeamWhiteListClaim {
				claimval, ok := m[k].(string)
				if !ok {
					log.Error("TeamWhiteList claim sent in openID user body cannot be casted as string")
					// continue auth with existing teammemberships for user
				}

				user.TeamMemberships = append(user.TeamMemberships, claimval)
				break
			}
		}
		log.Infof("team memberships present in user: %+v", user.TeamMemberships)
	}
	user.PrepareUserData()
	return nil
}
