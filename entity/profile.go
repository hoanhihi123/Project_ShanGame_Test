package entity

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/heroiclabs/nakama-common/runtime"
	pb "github.com/nakamaFramework/cgp-common/proto"
)

type ListProfile []*pb.SimpleProfile

// switch listProfile to map
// assign key = id_profile, value = object of profile
func (l ListProfile) ToMap() map[string]*pb.SimpleProfile {
	mapProfile := make(map[string]*pb.SimpleProfile)
	for _, p := range l {
		mapProfile[p.GetUserId()] = p
	}
	return mapProfile
}

// update info of profile if userId exists
// return info of ListProfile with list userID truyen vao
func GetProfileUsers(ctx context.Context, nk runtime.NakamaModule, userIDs ...string) (ListProfile, error) {
	accounts, err := nk.AccountsGetId(ctx, userIDs) // call api to get list account by list userId
	if err != nil {
		return nil, err
	}
	listProfile := make(ListProfile, 0, len(accounts))
	for _, acc := range accounts {
		u := acc.GetUser()
		var metadata map[string]interface{}
		json.Unmarshal([]byte(u.GetMetadata()), &metadata) // convert metadata from JSON to format Map[string]
		profile := pb.SimpleProfile{
			UserId:      u.GetId(),
			UserName:    u.GetUsername(),
			DisplayName: u.GetDisplayName(),
			Status:      InterfaceToString(metadata["status"]),
			AvatarId:    InterfaceToString(metadata["avatar_id"]),
			VipLevel:    ToInt64(metadata["vip_level"], 0),
		}
		playingMatchJson := InterfaceToString(metadata["playing_in_match"])

		if playingMatchJson == "" {
			profile.PlayingMatch = nil
		} else { // if player have played , save  history data  to PlayingMatch
			profile.PlayingMatch = &pb.PlayingMatch{}
			json.Unmarshal([]byte(playingMatchJson), profile.PlayingMatch)
		}
		if acc.GetWallet() != "" { // get balance of user current
			wallet, err := ParseWallet(acc.GetWallet())
			if err == nil {
				profile.AccountChip = wallet.Chips // if have money save to AccountChip properies
			}
		}
		listProfile = append(listProfile, &profile) // add profile new
	}
	return listProfile, nil
}

// get info profile of user
func GetProfileUser(ctx context.Context, nk runtime.NakamaModule, userID string) (*pb.SimpleProfile, error) {
	listProfile, err := GetProfileUsers(ctx, nk, userID)
	if err != nil {
		return nil, err
	}
	if len(listProfile) == 0 {
		return nil, errors.New("profile not found")
	}
	return listProfile[0], nil
}
