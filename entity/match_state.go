package entity

import (
	"fmt"
	"log"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	pb "github.com/nakamaFramework/cgp-common/proto"
)

const (
	MinPresences  = 1
	MaxPresences  = 7
	MinBetAllowed = 1
	MaxBetAllowed = 200
)

type MatchState struct {
	baseMatchState

	allowBet       bool
	allowInsurance bool
	allowAction    bool
	visited        map[string]bool // trường này dùng để làm gì
	userBets       map[string]*pb.ShanGamePlayerBet
	userLastBets   map[string]int64
	userHands      map[string]*Hand
	dealerHand     *Hand
	currentTurn    string
	currentHand    pb.ShanGameHandN0 // trường này dùng để làm gì
	gameState      pb.GameState
	updateFinish   *pb.ShanGameUpdateFinish
	POT            int64 // nơi chứa tiền cược user và tiền user nhận được sau thắng
	playerIsDealer string
	moneyOfServer  int64
	deck           *Deck // tôi muốn mỗi 1 bộ bài tương ứng với 1 trận đấu
}

// create new match
func NewMatchState(label *MatchLabel) MatchState {
	return MatchState{
		baseMatchState: baseMatchState{
			Label:               label, // xác định loại trận đấu
			MinPresences:        MinPresences,
			MaxPresences:        MaxPresences,
			Presences:           linkedhashmap.New(), // người chơi có mặt tại trận đấu
			PlayingPresences:    linkedhashmap.New(), // người chơi đang tham gia trận đấu
			LeavePresences:      linkedhashmap.New(), // người chơi đã rời khỏi trận đấu
			PresencesNoInteract: make(map[string]int, 0),
			balanceResult:       nil,
		},
		allowBet:    false,
		allowAction: false,

		userBets:       make(map[string]*pb.ShanGamePlayerBet, 0), // tiền cược của các player
		userLastBets:   make(map[string]int64, 0),
		userHands:      make(map[string]*Hand, 0), // những lá bài được chia cho player
		dealerHand:     &Hand{},                   // những lá bài được chia cho dealer
		currentTurn:    "",
		currentHand:    pb.ShanGameHandN0_SHANGAME_HAND_1ST, // bộ bài số mấy của player
		gameState:      pb.GameState_GameStateIdle,          // trạng thái trò chơi (start or end)
		updateFinish:   nil,                                 // update trạng thái trận đấu
		playerIsDealer: "",
		deck:           NewDeck(),
	}
}

// init info of user bets, user hands
func (s *MatchState) Init() {
	for k := range s.userBets {
		delete(s.userBets, k)
	}
	for k := range s.userHands {
		delete(s.userHands, k)
	}
	s.balanceResult = nil
	s.dealerHand = &Hand{
		first: make([]*pb.Card, 0),
	}
	s.currentTurn = ""
	s.updateFinish = nil
	s.currentHand = pb.ShanGameHandN0_SHANGAME_HAND_1ST
}

func (s *MatchState) InitVisited() {
	s.visited = make(map[string]bool, 0)
	for k := range s.userHands {
		s.visited[k] = false
	}
}

func (s *MatchState) IsAllVisited() bool {
	if s.visited == nil {
		return false
	} else {
		for _, v := range s.visited {
			if !v {
				return false
			}
		}
		return true
	}
}

func (s *MatchState) SetVisited(userId string) {
	s.visited[userId] = true
}

func (s *MatchState) SetCurrentHandN0(v pb.ShanGameHandN0) { s.currentHand = v }
func (s *MatchState) GetCurrentHandN0() pb.ShanGameHandN0  { return s.currentHand }

func (s *MatchState) SetCurrentTurn(v string) { s.currentTurn = v }
func (s *MatchState) GetCurrentTurn() string  { return s.currentTurn }

func (s *MatchState) GetGameState() pb.GameState  { return s.gameState }
func (s *MatchState) SetGameState(v pb.GameState) { s.gameState = v }

func (s *MatchState) GetPlayerHand(userId string) *pb.ShanGamePlayerHand {
	return s.userHands[userId].ToPb()
}
func (s *MatchState) GetPlayerPartOfHand(userId string, pos pb.ShanGameHandN0) *pb.ShanGameHand {
	return s.userHands[userId].ToPb().First
}

func (s *MatchState) GetDealerHand() *pb.ShanGamePlayerHand {
	return s.dealerHand.ToPb()
}

// divide more card
func (s *MatchState) AddCards(cards []*pb.Card, userId string, handN0 pb.ShanGameHandN0) {
	if userId == "" {
		s.dealerHand.AddCards(cards)
	} else {
		if _, found := s.userHands[userId]; !found {
			s.userHands[userId] = &Hand{
				userId: userId,
				first:  make([]*pb.Card, 0),
			}
		}
		s.userHands[userId].AddCards(cards)
	}
}

func (s *MatchState) SetAllowBet(v bool) { s.allowBet = v }
func (s *MatchState) IsAllowBet() bool   { return s.allowBet }

func (s *MatchState) SetAllowInsurance(v bool) { s.allowInsurance = v }
func (s *MatchState) IsAllowInsurance() bool   { return s.allowInsurance }

func (s *MatchState) SetAllowAction(v bool) { s.allowAction = v }
func (s *MatchState) IsAllowAction() bool   { return s.allowAction }

func (s *MatchState) SetUpdateFinish(v *pb.ShanGameUpdateFinish) { s.updateFinish = v }
func (s *MatchState) GetUpdateFinish() *pb.ShanGameUpdateFinish  { return s.updateFinish }

func (s *MatchState) GetUserBetById(userId string) *pb.ShanGamePlayerBet { return s.userBets[userId] }

// func (s *MatchState) CalChipForUserById(userId string, betValue int) {
// 	s.pot[userId].First += s.pot[userId].First + int64(betValue)
// }

// func (s *MatchState) SubstractChipOfUserById(userId string, betValue int) {
// 	// kiem tra xem du so du de tru ko
// 	if s.pot[userId].First >= int64(betValue) {
// 		s.pot[userId].First = s.pot[userId].First - int64(betValue)
// 	}
// 	s.pot[userId].First = 0
// }

// check can bet ?
func (s *MatchState) IsCanBet(userId string, balance int64, bet *pb.ShanGameBet) bool {
	if bet.Chips+s.userBets[userId].First+s.userBets[userId].Insurance+s.userBets[userId].Second > int64(MaxBetAllowed*s.Label.Bet) {
		return false
	}
	if balance < bet.Chips {
		return false
	}
	return true
}

// check lại điều kiện, so sánh mức cược với tiền mà player có
// mức cược của user < min bet => true => user ko được tham gia match
func (m *MatchState) IsBetFitMatch(userId string) bool {
	return m.userBets[userId].First < int64(m.Label.Bet)
}

// add bet for user
func (s *MatchState) AddBet(v *pb.ShanGameBet) {

	if _, found := s.userBets[v.UserId]; !found { // không tìm thấy userId trong userBet thì set value các trường = 0
		s.userBets[v.UserId] = &pb.ShanGamePlayerBet{
			UserId:    v.UserId,
			Insurance: 0,
			First:     0,
		}
	}
	fmt.Println("UserID current đặt cược: ", v.UserId, ", Mức cược của user current = ", s.userBets[v.UserId].First)

	// set mức cược cho player
	s.userBets[v.UserId].First += v.Chips
	s.userLastBets[v.UserId] = s.userBets[v.UserId].First
	s.allowAction = false

	if value, ok := s.PlayingPresences.Get(v.UserId); ok {
		playingPresence := value.(MyPrecense)
		playingPresence.Chips -= v.Chips // trừ đi tiền trong ví tạm của player tương ứng
	} else {
		// Xử lý trường hợp `User1` không tồn tại (nếu cần)
		fmt.Println(v.UserId, " không tồn tại trong danh sách Presences")
	}

	fmt.Println("UserID current sau khi tăng mức đặt cược: ", v.UserId, ", Mức cược của user current sau khi đặt cược= ", s.userBets[v.UserId].First)
}

func (s *MatchState) GetBet(v *pb.ShanGameBet) int64 {
	return s.userBets[v.UserId].First
}

// add bet for user
func (s *MatchState) ReduceBet(v *pb.ShanGameBet) {
	if _, found := s.userBets[v.UserId]; !found {
		s.userBets[v.UserId] = &pb.ShanGamePlayerBet{
			UserId:    v.UserId,
			Insurance: 0,
			First:     0,
			Second:    0,
		}
	}
	fmt.Println("UserID current đặt cược: ", v.UserId, ", Mức cược của user current = ", s.userBets[v.UserId].First)

	s.userBets[v.UserId].First -= v.Chips
	s.userLastBets[v.UserId] = s.userBets[v.UserId].First
	s.allowAction = false

	if value, ok := s.PlayingPresences.Get(v.UserId); ok {
		playingPresence := value.(MyPrecense)
		playingPresence.Chips += v.Chips // cộng lại tiền trong ví tạm của player tương ứng
	} else {
		// Xử lý trường hợp `User1` không tồn tại (nếu cần)
		fmt.Println(v.UserId, " không tồn tại trong danh sách Presences")
	}

	fmt.Println("UserID current sau khi giảm mức đặt cược: ", v.UserId, ", Mức cược của user current sau khi đặt cược= ", s.userBets[v.UserId].First)

}

func (s *MatchState) IsCanInsuranceBet(userId string, balance int64) bool {
	return balance*2 >= s.userBets[userId].First
}

func (s *MatchState) InsuranceBet(userId string) int64 {
	s.userBets[userId].Insurance = s.userBets[userId].First / 2
	return s.userBets[userId].Insurance
}

func (s *MatchState) IsCanDoubleDownBet(userId string, balance int64, pos pb.ShanGameHandN0) bool {
	return balance >= s.userBets[userId].First
}

func (s *MatchState) DoubleDownBet(userId string, pos pb.ShanGameHandN0) int64 {
	r := int64(0)
	if pos == pb.ShanGameHandN0_SHANGAME_HAND_1ST {
		r = s.userBets[userId].First
		s.userBets[userId].First *= 2
	}
	return r
}

// func (s *MatchState) IsCanSplitHand(userId string, balance int64) bool {
// 	if balance >= s.userBets[userId].First {
// 		return s.userHands[userId].PlayerCanSplit()
// 	}
// 	return false
// }

// func (s *MatchState) SplitHand(userId string) int64 {
// 	s.userBets[userId].Second = s.userBets[userId].First
// 	s.userHands[userId].Split()
// 	return s.userBets[userId].Second
// }

func (s *MatchState) Rebet(userId string) int64 {
	s.userBets[userId].First = s.userLastBets[userId]
	return s.userLastBets[userId]
}

func (s *MatchState) DoubleBet(userId string) int64 {
	if _, found := s.userBets[userId]; found && s.userBets[userId].First >= MinBetAllowed*int64(s.Label.Bet) {
		r := s.userBets[userId].First
		s.userBets[userId].First *= 2
		s.userLastBets[userId] = s.userBets[userId].First
		return r
	} else if _, found := s.userLastBets[userId]; found {
		if _, found := s.userBets[userId]; !found {
			s.userBets[userId] = &pb.ShanGamePlayerBet{
				UserId:    userId,
				Insurance: 0,
				First:     0,
				Second:    0,
			}
		}
		s.userLastBets[userId] *= 2
		s.userBets[userId].First = s.userLastBets[userId]
		return s.userLastBets[userId]
	}
	return 0
}

func (s *MatchState) IsCanRebet(userId string, balance int64) bool {
	if _, found := s.userBets[userId]; found {
		return false
	}
	if _, found := s.userLastBets[userId]; !found || s.userLastBets[userId] > balance {
		return false
	}
	return true
}

func (s *MatchState) IsCanDoubleBet(userId string, balance int64) bool {
	if _, found := s.userBets[userId]; found {
		if s.userBets[userId].First > balance {
			return false
		} else {
			return true
		}
	} else if _, found := s.userLastBets[userId]; found && s.userLastBets[userId]*2 <= balance {
		return true
	}
	return false
}

func (s *MatchState) IsCanHit(userId string, pos pb.ShanGameHandN0) bool {
	return s.userHands[userId].PlayerCanDraw(pos)
}

// userBets : danh sách những người dùng đã đặt cược
// s.userBets[userId].First: tại thông số first
// nếu giá trị first > 0 , tức là số lần đặt cược của user đó
func (s *MatchState) IsBet(userId string) bool {
	if _, found := s.userBets[userId]; found && s.userBets[userId].First > 0 {
		return true
	}
	return false
}

func (s *MatchState) CalcGameFinish() *pb.ShanGameUpdateFinish {
	result := &pb.ShanGameUpdateFinish{
		BetResults: make([]*pb.ShanGamePLayerBetResult, 0),
	}
	for _, h := range s.userHands {
		result.BetResults = append(result.BetResults, s.getPlayerBetResult(h.userId))
	}
	return result
}

// lấy kết quả cược của người chơi
func (s *MatchState) getPlayerBetResult(userId string) *pb.ShanGamePLayerBetResult {
	defer func() { s.userBets[userId].Insurance = 0 }()
	userBet := s.userBets[userId]
	r1 := s.userHands[userId].Compare(s.dealerHand)
	first := &pb.ShanGameBetResult{
		BetAmount: userBet.First,
		WinAmount: 0,
		Total:     userBet.First,
	}

	if first.BetAmount > 0 {
		first.IsWin = int32(r1)
		if r1 > 0 {
			first.WinAmount = first.BetAmount
			first.Total = first.BetAmount + first.WinAmount
		} else if r1 < 0 {
			first.WinAmount = -first.BetAmount
			first.Total = first.BetAmount + first.WinAmount
		}
	}

	return &pb.ShanGamePLayerBetResult{
		UserId: userId,
		First:  first,
	}
}

func (s *MatchState) GetLegalActions() []pb.ShanGameActionCode {
	result := make([]pb.ShanGameActionCode, 0)
	if s.userHands[s.currentTurn].PlayerCanDraw(s.currentHand) {
		result = append(result, pb.ShanGameActionCode_SHANGAME_ACTION_HIT)

		result = append(result, pb.ShanGameActionCode_SHANGAME_ACTION_STAY)
	}
	return result
}

func (s *MatchState) DealerPotentialBlackjack() bool {
	return s.dealerHand.DealerPotentialBlackjack()
}

func (s *MatchState) IsDealerMustDraw() bool {
	return s.dealerHand.DealerMustDraw()
}

func (s *MatchState) IsGameEnded() bool {
	return s.updateFinish != nil
}

func (s *MatchState) GetPlayerChipsInWallet(userID string) int64 {
	if value, ok := s.PlayingPresences.Get(userID); ok {
		playing_presence := value.(MyPrecense)
		return playing_presence.Chips
	}
	return 0
}

// set money of player in wallet
func (s *MatchState) SetUserChipsInWallet(userID string, moneyChange int64) {
	if value, ok := s.PlayingPresences.Get(userID); ok {
		playing_presence := value.(MyPrecense) // Ép kiểu
		playing_presence.Chips = moneyChange   // Cập nhật giá trị Chips của player lose
		s.Presences.Put(userID, playing_presence)
	}
}

// xác định ai thắng thua
func (s *MatchState) CalPointFor_Player_Dealer() {
	fmt.Println("Xét xem player or dealer win / lose ....")
	// list userId is win
	userId_result_Win := map[string]int64{}

	if len(s.userHands) > 0 && s.userHands != nil {
		fmt.Println("Chạy vào trong sau khi (len(s.userHands)>0) bắt đầu so sánh player và dealer....")
		for _, userHand := range s.userHands {
			s.gameState = pb.GameState_GameStateReward // trận đấu status: được tính điểm
			result := int(0)
			if s.dealerHand != nil {
				result = userHand.Compare(s.dealerHand) // result after compare each player - dealer
			}
			tiLeAn_dealer := s.dealerHand.GetTiLeThangThuaPlayer() // get ti le thang, thua
			tiLeAn_player := userHand.GetTiLeThangThuaPlayer()
			moneyBet_player := s.userBets[userHand.userId].First        // mức cược của player
			moneyBet_dealer := s.GetBetOfUser_byID(s.dealerHand.userId) // mức tiền cược dealer

			if result == 1 { // player win, dealer lose
				fmt.Println("Player: " + userHand.userId + " - WIN ^_^")

				if s.playerIsDealer != "" { // player is dealer => cần tính lại, sau khi lấy được danh sách các player_win
					fmt.Println("Trường hợp playerIsDealer, nên tính tiền sẽ được tính sau....")
					userId_result_Win[userHand.userId] = int64(tiLeAn_player)
				} else {
					// option: dealer is server => chỉ cộng tiền cho user luôn
					fmt.Println("Trường hợp dealer là server, cộng tiền cho user luôn")
					chips_inWalletOfPlayer := s.GetPlayerChipsInWallet(userHand.userId)
					moneyPlayerWin := moneyBet_player * int64(tiLeAn_player)
					tienHoPlayer := s.GetTiLeTienHo_User(userHand.userId) * moneyPlayerWin
					s.CalculatorMoneyForUserWin(1, userHand.userId, int64(tiLeAn_player), (chips_inWalletOfPlayer + (moneyPlayerWin - tienHoPlayer)), moneyBet_player)
				}

			} else if result == -1 { // dealer win , player lose
				fmt.Println("Dealer : " + s.dealerHand.userId + " - WIN ^_^")

				// mức tiền cược player
				moneyDealerWin := moneyBet_dealer * int64(tiLeAn_dealer)

				// lấy tiền trong wallet of player
				chips_inWalletOfPlayer := s.GetPlayerChipsInWallet(userHand.userId)

				// check player có đủ tiền trả cho dealer không ?
				// check (tổng tiền trong wallet of user + tiền user đã cược)
				if (chips_inWalletOfPlayer + moneyBet_player) < moneyDealerWin { // dealer win //  tổng tiền player không đủ trả cho dealer

					s.CalculatorMoneyForUserWin(-1, userHand.userId, int64(tiLeAn_player), 0, 0)

				} else { // dealer win (tổng tiền của player đủ trả cho dealer)
					// sẽ có trường hợp  trong ví đủ tiền
					// ngược lại thì cần lấy tiền từ ván cược => thanh toán
					if chips_inWalletOfPlayer >= moneyDealerWin { // dealer win
						s.CalculatorMoneyForUserWin(-1, userHand.userId, int64(tiLeAn_player), (chips_inWalletOfPlayer - moneyDealerWin), moneyBet_player)
					}

					// trường hợp còn lại
					// dealer win, player không đủ tiền trong ví để trả server, cần lấy tiền cược để trả server
					s.CalculatorMoneyForUserWin(-1, userHand.userId, int64(tiLeAn_player), ((chips_inWalletOfPlayer + moneyBet_player) - moneyDealerWin), 0)

					// xóa player khỏi match
					s.PlayingPresences.Remove(userHand.userId)
				}
			} else {
				fmt.Println("Player:", userHand.userId, " and Dealer:", s.dealerHand.userId, " hòa nhau")
			}
		}
	}

	if len(userId_result_Win) > 0 { // player win && dealer is player
		isResultCheckSumMoney := s.Check_dealerIsPlayer_enoughMoney(userId_result_Win)

		// player is dealer
		if isResultCheckSumMoney { // dealer_is_player have enough money for pay player_win
			for _, userHand := range s.userHands {
				betOfPlayerInMatch := s.userBets[userHand.userId].First
				tiLeAn_player := userHand.GetTiLeThangThuaPlayer() // lấy tỉ lệ ăn player
				chips_inWalletOfPlayer := s.GetPlayerChipsInWallet(userHand.userId)

				// xem danh sach user = userId_result ?
				for key_userResult := range userId_result_Win {
					if key_userResult == userHand.userId {
						// tiền ăn của player
						moneyWinPlayer := betOfPlayerInMatch * int64(tiLeAn_player)

						// tính tiền hồ mà server nhận
						tienHoPlayer := s.GetTiLeTienHo_User(userHand.userId) * moneyWinPlayer

						s.CalculatorMoneyForUserWin(1, userHand.userId, int64(tiLeAn_player),
							(moneyWinPlayer + chips_inWalletOfPlayer - tienHoPlayer), betOfPlayerInMatch)
					}
				}
			}
		} else { // dealer không đủ tiền trả cho các player
			// tính tỉ lệ thắng của từng user và lấy tiền từ wallet of dealer trả lần lượt cho các player
			sumMoney_DealerPaid_Player := s.sumMoneyDealer_needToPayPlayer(userId_result_Win)
			sumMoney_DealerCurrent := s.POT
			moneyPlayerWin_fact := int64(0)
			moneyPlayerWin_tiLe := int64(0)
			tienHoPlayer := int64(0)

			for _, userHand := range s.userHands {
				// lấy tỉ lệ ăn player
				betOfPlayerInMatch := s.userBets[userHand.userId].First
				tiLeAn_player := userHand.GetTiLeThangThuaPlayer()

				// xem danh sach user = userId_result ?
				for key_userResult := range userId_result_Win {
					// lấy ra các user thắng dealer_isPlayer
					if key_userResult == userHand.userId {
						// tính tiền thắng cho từng player
						// thiết lập lại tiền cho player
						moneyPlayerWin_fact = int64(tiLeAn_player) * betOfPlayerInMatch
						// tiền userWin = (userWinFact / tổng tiền phải được nhận) * tổng tiền hiện còn của dealer
						moneyPlayerWin_tiLe = (moneyPlayerWin_fact / sumMoney_DealerPaid_Player) * sumMoney_DealerCurrent

						// tính tiền hồ mà server nhận
						tienHoPlayer = s.GetTiLeTienHo_User(key_userResult) * moneyPlayerWin_tiLe
						s.moneyOfServer += tienHoPlayer
						// update wallet of Player
						// set wallet of player = 0
						s.SetUserChipsInWallet(userHand.userId, moneyPlayerWin_tiLe-tienHoPlayer)

						// player is dealer
						s.POT -= moneyPlayerWin_tiLe

						// kiểm tra xem dealer còn đủ điều kiện làm dealer hay không ?
						if s.POT < int64(s.Label.Bet) {
							s.SetUserChipsInWallet(s.dealerHand.userId, 0)
							s.playerIsDealer = ""
							s.dealerHand = &Hand{} // nơi chứa bộ bài của dealer (là player/server đều sẽ set value vào đây)
						}
					}
				}
			}
		}
	}
}

// get tỉ lệ tiền hồ của mỗi player win
func (s *MatchState) GetTiLeTienHo_User(userID string) int64 {
	vipLevel := int64(0)     // vipLevel of user
	bankOfPlayer := int64(0) // total money of player
	tiLeTienCuoc := int64(0)
	// làm sao để lấy ra vip của player
	if value, ok := s.Presences.Get(userID); ok {
		presence := value.(MyPrecense)                           // Ép kiểu về MyPrecense
		vipLevel = presence.VipLevel                             // Gán giá trị Chips cho biến mới
		bankOfPlayer = presence.Chips + s.userBets[userID].First // ? có tính cả mức cược mà presence đã cược hay ko ? // may be , có , vì nếu wallet ko đủ => cũng phải dùng bet
		tiLeTienCuoc = (s.userBets[userID].First / presence.Chips) / 100
	}

	// lấy ra tỉ lệ cược của mỗi user tương ứng theo vipLevel
	if vipLevel == 0 {
		if (bankOfPlayer >= 0 && bankOfPlayer <= 100000000000) && (tiLeTienCuoc >= 0 && tiLeTienCuoc <= 100) {
			return 5
		}
	} else if vipLevel == 1 {
		if bankOfPlayer >= 0 && bankOfPlayer <= 100000000000 {
			if tiLeTienCuoc >= 0 && tiLeTienCuoc <= 60 {
				return 8
			} else if tiLeTienCuoc > 60 && tiLeTienCuoc <= 85 {
				return 12
			} else if tiLeTienCuoc > 85 && tiLeTienCuoc <= 100 {
				return 15
			}
		} else if bankOfPlayer <= 100000000000 {
			// eg (0-100), eg (0-60), eg (60-85), eg (85-100)

			if tiLeTienCuoc >= 0 && tiLeTienCuoc <= 60 {
				return 8
			} else if tiLeTienCuoc > 60 && tiLeTienCuoc <= 85 {
				return 15
			} else if tiLeTienCuoc > 85 && tiLeTienCuoc <= 100 {
				return 20
			}
		}
	} else if vipLevel <= 4 {
		if bankOfPlayer >= 0 && bankOfPlayer <= 2000000 {
			if tiLeTienCuoc >= 0 && tiLeTienCuoc <= 60 {
				return 6
			} else if tiLeTienCuoc > 60 && tiLeTienCuoc <= 70 {
				return 8
			} else if tiLeTienCuoc > 70 && tiLeTienCuoc <= 85 {
				return 15
			} else if tiLeTienCuoc <= 100 {
				return 12
			}
		} else if bankOfPlayer <= 100000000000 {
			if tiLeTienCuoc >= 0 && tiLeTienCuoc <= 60 {
				return 6
			} else if tiLeTienCuoc > 60 && tiLeTienCuoc <= 70 {
				return 10
			} else if tiLeTienCuoc > 70 && tiLeTienCuoc <= 85 {
				return 15
			} else if tiLeTienCuoc <= 100 {
				return 20
			}
		}
	} else if vipLevel <= 10 {
		if bankOfPlayer >= 0 && bankOfPlayer <= 5000000 {
			if tiLeTienCuoc >= 0 && tiLeTienCuoc <= 60 {
				return 6
			} else if tiLeTienCuoc > 60 && tiLeTienCuoc <= 70 {
				return 8
			} else if tiLeTienCuoc > 70 && tiLeTienCuoc <= 85 {
				return 10
			} else if tiLeTienCuoc <= 100 {
				return 12
			}
		} else if bankOfPlayer <= 100000000000 {
			if tiLeTienCuoc >= 0 && tiLeTienCuoc <= 60 {
				return 6
			} else if tiLeTienCuoc > 60 && tiLeTienCuoc <= 70 {
				return 10
			} else if tiLeTienCuoc > 70 && tiLeTienCuoc <= 85 {
				return 15
			} else if tiLeTienCuoc <= 100 {
				return 20
			}
		}
	}

	return 0

}

// check sum money dealer_is_player need to pay for player
func (s *MatchState) Check_dealerIsPlayer_enoughMoney(userId_result_Win map[string]int64) bool {
	sum_moneyDealerNeed_paid := int64(0)
	if len(userId_result_Win) > 0 {
		for _, userHand := range s.userHands {
			s.gameState = pb.GameState_GameStateReward // trận đấu status: được tính điểm
			result := userHand.Compare(s.dealerHand)
			// lấy tỉ lệ ăn player
			tiLeAn_player := userHand.GetTiLeThangThuaPlayer()
			// xem danh sach user = userId_result ?
			for key_userResult := range userId_result_Win {
				if result == -1 && key_userResult == userHand.userId {
					// tinh tong tien cua dealer
					sum_moneyDealerNeed_paid += int64(tiLeAn_player) * s.userBets[key_userResult].First
				}
			}
		}
	}

	if sum_moneyDealerNeed_paid > s.POT {
		return false
	}
	return true
}

// player win dealer, but dealer not enough money to pay for each player win
// sum money dealer_is_player need to pay for player
func (s *MatchState) sumMoneyDealer_needToPayPlayer(userId_result_Win map[string]int64) int64 {
	sum_moneyDealerNeed_paid := int64(0)
	if len(userId_result_Win) > 0 {
		for _, userHand := range s.userHands {
			// lấy tỉ lệ ăn player
			tiLeAn_player := userHand.GetTiLeThangThuaPlayer()
			// xem danh sach user = userId_result ?
			for key_userResult := range userId_result_Win {
				if key_userResult == userHand.userId {
					// tinh tong tien cua dealer
					sum_moneyDealerNeed_paid += int64(tiLeAn_player) * s.userBets[userHand.userId].First
				}
			}
		}
	}

	return sum_moneyDealerNeed_paid
}

func (s *MatchState) GetBetOfUser_byID(userId string) int64 { return s.userBets[userId].First }

// -1 player lose, 1 player win, 0 hoa nhau
func (s *MatchState) CalculatorMoneyForUserWin(
	statusMatch int, userID_player string, tiLeAn_player int64,
	setChipPlayer int64, setBetOfPlayer int64) {

	moneyBet_dealer := s.GetBetOfUser_byID(s.dealerHand.userId) // mức tiền cược dealer
	moneyBet_player := s.GetBetOfUser_byID(userID_player)       // mức tiền cược player

	tiLeAn_dealer := s.dealerHand.GetTiLeThangThuaPlayer() // get ti le thang, thua
	// tiLeAn_player := s.userHands.GetTiLeThangThuaPlayer()

	moneyDealerWin := moneyBet_dealer * int64(tiLeAn_dealer)
	moneyPlayerWin := moneyBet_player * int64(tiLeAn_player)

	tienHoPlayer := s.GetTiLeTienHo_User(userID_player) * moneyPlayerWin

	// lấy tiền trong wallet of player
	chips_inWalletOfPlayer := s.GetPlayerChipsInWallet(userID_player) // get chip from playingPresence

	if statusMatch == -1 { // dealer win
		// tiền thắng của dealer chưa chắc = mức cược dealer * tỉ lệ thắng dealer , vì player chưa chắc đủ tiền trả
		if moneyBet_player+chips_inWalletOfPlayer < moneyDealerWin {
			moneyDealerWin = moneyBet_player + chips_inWalletOfPlayer
		}

		// moneyDealerWin = (chips_inWalletOfPlayer + moneyBet_player) * tiLeAn_player // all money player lose have
		// set chip in wallet of player = 0
		s.SetUserChipsInWallet(userID_player, setChipPlayer) // set chip at playingPresence

		// set bet of player in match = 0
		s.userBets[userID_player].First = setBetOfPlayer

		// xóa player khỏi match
		if moneyBet_player < int64(s.Label.Bet) {
			s.PlayingPresences.Remove(userID_player)
		}
		// chưa set trường hợp nếu dealer lose => bị trừ tiền, có thể ko đủ điều kiện làm dealer nữa

		// tính tiền hồ mà server nhận
		tienHoDealer := s.GetTiLeTienHo_User(s.dealerHand.userId) * moneyDealerWin
		s.moneyOfServer += int64(tienHoDealer)

		// set money condition Dealer is player
		if s.playerIsDealer != "" {
			s.POT += (moneyDealerWin - int64(tienHoDealer))
		} else { // dealer is server
			s.moneyOfServer += (moneyDealerWin - int64(tienHoDealer))
		}
	} else if statusMatch == 1 { // player win
		// - server ( chỉ cộng tiền cho player thông thường + tính tiền hồ cho server )
		// - player is dealer ( cộng tiền cho player + trừ tiền của player từ POT + tính tiền hồ cho server )

		// cộng tiền hồ cho server
		s.moneyOfServer += int64(tienHoPlayer)

		// set bet of player in match
		s.userBets[userID_player].First = setBetOfPlayer

		// cộng tiền ăn của player vào wallet of player
		s.SetUserChipsInWallet(userID_player, (chips_inWalletOfPlayer + moneyPlayerWin - tienHoPlayer))

		// trừ tiền của server nếu player is dealer
		if s.playerIsDealer != "" { // player is dealer , dealer is server thì ko bị trừ tiền
			s.POT -= moneyPlayerWin
		}

	}
}

func (s *MatchState) chiaBaiChoPlayerTuongUng() {
	if len(s.userBets) > 0 {
		for _, userBet := range s.userBets { // lấy danh sách userBet - lấy ra userID tham gia
			fmt.Println("Chia bài cho user hiện tại có id: " + userBet.UserId)
			listCard_chia2La, err := s.deck.Deal(2) // mỗi userBet - lấy ra 2 lá bài trong bộ bài

			if err != nil {
				log.Fatal("Lỗi khi chia bài : ", err, " , khi chia bài cho user: ", userBet.UserId)
			}

			fmt.Printf("Danh sách lá bài được rút: %+v", listCard_chia2La.Cards)
			// chưa check trường hợp userBet là userHand của PlayerIsDealer
			// User1 tại sao set value lại lỗi do chưa khởi tạo giá trị từ trước ?

			// chia bài cho dealerIsPlayer or serverIsDealer
			if userBet.UserId == s.playerIsDealer { // chia bài cho dealer: chia 1 or 2 lá tương ứng cho dealer
				// trường hợp chưa được chia lá bài nào
				// check chia max <=3
				fmt.Println("\nChia bài cho dealer, với id = " + userBet.UserId)
				if len(s.dealerHand.first) == 0 && s.dealerHand.userId == "" { // dealerhand chưa được khởi tạo và thêm giá trị nào
					s.dealerHand = &Hand{
						userId: userBet.UserId,
						first:  listCard_chia2La.Cards,
					}
				} else if len(s.dealerHand.first) <= 3 && s.dealerHand.userId == "" {
					listCard_addOneCard, err := s.deck.Deal(1)
					if err != nil {
						log.Fatal("Lỗi khi chia bài : ", err, " , khi chia bài cho user: ", userBet.UserId)
						continue
					} else {
						s.dealerHand.first = append(s.dealerHand.first, listCard_addOneCard.Cards...)
					}
				}
				fmt.Printf("Bộ bài của dealer sau khi chia bài: %+v\n", s.dealerHand.first)
			} else { // chia bài cho player
				fmt.Println("\nChia bài cho player, với id = " + userBet.UserId)
				if len(s.userHands) > 0 {
					isCheckUserId_existInUserHand := true  // check userId đã có trong userHand chưa
					for _, userHand := range s.userHands { // tại sao nó chạy vào đây là lỗi ?
						if len(userHand.first) == 2 && userHand.userId == userBet.UserId {
							isCheckUserId_existInUserHand = false
							listCard_addOneCard, err := s.deck.Deal(1)
							if err != nil {
								log.Fatal("Lỗi khi chia bài : ", err, " , khi chia bài cho user: ", userBet.UserId)
								continue
							} else {
								userHand.first = append(userHand.first, listCard_addOneCard.Cards...)
							}
							break
						}

					}

					if isCheckUserId_existInUserHand { // userId chưa tồn tại trong UserHand
						// set thông tin bài chia được vào userHands
						s.userHands[userBet.UserId] = &Hand{
							userId: userBet.UserId,
							first:  listCard_chia2La.Cards,
						}
					}
				} else { // trường hợp userHand chưa có chứa thông tin của bất kỳ ai => thêm mới 1 userHand tại userId hiện tại
					s.userHands[userBet.UserId] = &Hand{
						userId: userBet.UserId,
						first:  listCard_chia2La.Cards,
					}
				}
				fmt.Printf("Bộ bài của player sau khi chia bài: %+v\n", s.userHands[userBet.UserId].first)
			}
		}
	} else {
		fmt.Println("Message thông báo: Chưa có user nào đặt cược vào ván chơi!")
		return
	}
}