package entity

import (
	"context"
	"math"
	"time"

	pb "github.com/nakamaFramework/cgp-common/proto"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/heroiclabs/nakama-common/runtime"
)

type MatchLabel struct {
	Open         int32  `json:"open"`
	Bet          int32  `json:"bet"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	Password     string `json:"password"`
	MaxSize      int32  `json:"max_size"`
	MockCodeCard int32  `json:"mock_code_card"`
}

type baseMatchState struct {
	Label               *MatchLabel
	MinPresences        int
	MaxPresences        int
	Presences           *linkedhashmap.Map
	PlayingPresences    *linkedhashmap.Map
	LeavePresences      *linkedhashmap.Map
	PresencesNoInteract map[string]int
	JoinsInProgress     int
	CountDownReachTime  time.Time
	LastCountDown       int
	balanceResult       *pb.BalanceResult
}

func (s *baseMatchState) GetBalanceResult() *pb.BalanceResult {
	return s.balanceResult
}

func (s *baseMatchState) SetBalanceResult(u *pb.BalanceResult) {
	s.balanceResult = u
}

func (s *baseMatchState) ResetBalanceResult() {
	s.balanceResult = nil
}

func (s *baseMatchState) SetUpCountDown(d time.Duration) {
	s.CountDownReachTime = time.Now().Add(d)
	s.LastCountDown = 1
}
func (s *baseMatchState) SetLastCountDown(v int) { s.LastCountDown = v }

func (s *baseMatchState) GetRemainCountDown() float64 {
	return time.Until(s.CountDownReachTime).Seconds()
}

func (s *baseMatchState) IsNeedNotifyCountDown() bool {
	return s.LastCountDown == -1 || int(math.Round(s.GetRemainCountDown())) != s.LastCountDown
}

func (s *baseMatchState) IsReadyToPlay() bool { return s.Presences.Size() >= s.MinPresences }

func (s *baseMatchState) GetPresenceSize() int { return s.Presences.Size() }

func (s *baseMatchState) AddPresence(ctx context.Context,
	nk runtime.NakamaModule,
	presences []runtime.Presence,
) {
	for _, presence := range presences {
		m := NewMyPrecense(ctx, nk, presence)
		s.Presences.Put(presence.GetUserId(), m)
		s.ResetUserNotInteract(presence.GetUserId())
	}
}

func (s *baseMatchState) RemovePresences(presences ...runtime.Presence) {
	for _, p := range presences {
		s.Presences.Remove(p.GetUserId())
		delete(s.PresencesNoInteract, p.GetUserId())
	}
}

func (s *baseMatchState) GetPresence(userId string) runtime.Presence {
	_, v := s.Presences.Find(func(key, value interface{}) bool { return key == userId })
	if v != nil {
		return v.(runtime.Presence)
	} else {
		return nil
	}
}

func (s *baseMatchState) GetPresences() []runtime.Presence {
	p := make([]runtime.Presence, 0)
	s.Presences.Each(func(key, value interface{}) { p = append(p, value.(runtime.Presence)) })
	return p
}

func (s *baseMatchState) SetupMatchPresence() {
	s.PlayingPresences = linkedhashmap.New()
	p := make([]runtime.Presence, 0, s.GetPresenceSize())
	s.Presences.Each(func(key, value interface{}) { p = append(p, value.(runtime.Presence)) })
	s.AddPlayingPresences(p...)
}

func (s *baseMatchState) AddPlayingPresences(presences ...runtime.Presence) {
	for _, p := range presences {
		k := p.GetUserId()
		s.PlayingPresences.Put(k, p)
		if v, exist := s.PresencesNoInteract[k]; exist {
			s.PresencesNoInteract[k] = v + 1
		} else {
			s.PresencesNoInteract[k] = 1
		}
	}
}

func (s *baseMatchState) GetPlayingPresences() []runtime.Presence {
	presences := make([]runtime.Presence, 0)
	s.PlayingPresences.Each(func(key interface{}, value interface{}) {
		presences = append(presences, value.(runtime.Presence))
	})

	return presences
}

func (s *baseMatchState) AddLeavePresence(presences ...runtime.Presence) {
	for _, presence := range presences {
		s.LeavePresences.Put(presence.GetUserId(), presence)
	}
}

func (s *baseMatchState) ApplyLeavePresence() {
	s.LeavePresences.Each(func(key, value interface{}) {
		s.Presences.Remove(key)
		delete(s.PresencesNoInteract, key.(string))
	})
	s.LeavePresences = linkedhashmap.New()
}

func (s *baseMatchState) RemoveLeavePresence(userId string) {
	s.LeavePresences.Remove(userId)
}

func (s *baseMatchState) GetLeavePresences() []runtime.Presence {
	presences := make([]runtime.Presence, 0)
	s.LeavePresences.Each(func(key interface{}, value interface{}) {
		presences = append(presences, value.(runtime.Presence))
	})

	return presences
}

func (s *baseMatchState) GetPresenceNotInteract(roundGame int) []runtime.Presence {
	listPresence := make([]runtime.Presence, 0)
	s.Presences.Each(func(key interface{}, value interface{}) {
		if roundGameNotInteract, exist := s.PresencesNoInteract[key.(string)]; exist && roundGameNotInteract >= roundGame {
			listPresence = append(listPresence, value.(runtime.Presence))
		}
	})
	return listPresence
}

func (s *baseMatchState) ResetUserNotInteract(userId string) {
	s.PresencesNoInteract[userId] = 0
}
