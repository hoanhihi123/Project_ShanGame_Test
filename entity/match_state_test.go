package entity

import (
	"fmt"
	"testing"

	pb "github.com/nakamaFramework/cgp-common/proto"
)

/*
cac truong hop co the check

	server = dealer
	user = dealer
	user = join when user other join game

	deal card and manage card
	cal point and return player win

	end game
*/
func TestMatchState_hoan(t *testing.T) {

	fmt.Println("Start game...")
	s := NewMatchState(&MatchLabel{
		Open:     2,
		Bet:      1000,
		Code:     "test",
		Name:     "test_table",
		Password: "",
		MaxSize:  7,
	})
	fmt.Printf("MatchState sau khi khởi tạo: %+v", s)

	minBetToPlayMatch := s.Label.Bet
	fmt.Println("\nMức cược tối thiểu :", minBetToPlayMatch)

	s.playerIsDealer = "" // server is dealer

	// khoi tao match
	fmt.Println("khởi tạo trận đấu .... ")
	// user request vào game

	s.Presences.Put("User1", MyPrecense{
		Chips:    100000,
		VipLevel: 1,
	})
	s.Presences.Put("User2", MyPrecense{
		Chips:    20000,
		VipLevel: 3,
	})

	// kiểm tra xem user có đủ điều kiện làm Dealer hay không ?
	// giả sử player muốn làm dealer

	s.Presences.Put("User5_dealer", MyPrecense{ // info of player : id, amountChips
		Chips:    200000,
		VipLevel: 5,
	})

	// set trường hợp, server đang là dealer , player user5 xin làm dealer
	s.playerIsDealer = ""
	// 	trước khi vào trận đấu  - player xin làm dealer
	if s.playerIsDealer == "" { // server is  dealer

		// lấy chips của user muốn xin làm dealer
		if value, ok := s.Presences.Get("User5_dealer"); ok {
			presence := value.(MyPrecense)
			if presence.Chips > int64(minBetToPlayMatch) { // so sánh với mức cược tối thiểu của trận đấu
				s.playerIsDealer = "User5_dealer"
				s.POT = presence.Chips // set banker = tổng tiền trong ví của player đang có
				s.dealerHand = &Hand{
					userId: "User5_dealer",
				}
			} else { // server is dealer
				if value, ok := s.Presences.Get(""); ok {
					presence := value.(MyPrecense)

					s.playerIsDealer = ""
					s.POT = presence.Chips // set chips của server đặt cược vào POT
					s.dealerHand = &Hand{
						userId: "",
					}
				}
			}
		}
	} else { // player is dealer - player khác xin làm dealer thay thế

		// lấy mức chip của player đang là dealer
		chip_playerIsDealer := int64(0)
		if value, ok := s.Presences.Get(s.playerIsDealer); ok {
			precence := value.(MyPrecense)
			chip_playerIsDealer = precence.Chips
		}

		// lấy mức chip của player khác muốn xin làm dealer thay thế
		chip_player2 := int64(0)
		userId_player2 := "User5_dealer"
		if value, ok := s.Presences.Get(userId_player2); ok {
			precence := value.(MyPrecense)
			chip_player2 = precence.Chips
		}

		// check điều kiện đủ để player khác xin làm dealer - thay thế dealer current
		if chip_player2 > chip_playerIsDealer && chip_player2 > int64(minBetToPlayMatch) {
			s.playerIsDealer = userId_player2
			s.POT = chip_player2 // set banker = tổng tiền trong ví mà dealer hiện có
			s.dealerHand = &Hand{
				userId: userId_player2,
			}
		}
	}

	fmt.Println("Server or Player is Dealer ? ", s.playerIsDealer, ", \t Pot = ", s.POT)

	fmt.Println("DealerHand current: ", s.dealerHand.userId)
	fmt.Println("UserHand current have size: ", len(s.userHands))

	s.Init()
	// set info of player with fake precencese
	fmt.Println("khởi tạo các user join vào trận đấu để chơi .... ")

	// khi nào thì check maxPresence ?
	if s.Presences.Size() > 0 {
		i := s.Presences.Iterator()
		for i.Next() {
			// check size match
			if s.PlayingPresences.Size() > 7 {
				fmt.Println("Match is full, Can't add more player ... ")
				break
			}
			// check trường hợp có add server vào game hay không?
			if i.Key() == "" && s.playerIsDealer != "" { // trường hợp server là player, và isPlayingDealer có giá trị idPlayer => không thêm server vào ván game
				continue
			}
			s.PlayingPresences.Put(i.Key(), i.Value())
		}
	} else {
		fmt.Println("Hiện tại không có người chơi nào, cần ít nhất 1 user để chơi game!")
		return
	}

	// them muc cuoc cho user ( chon thong thuong, + - )
	fmt.Println("Các user chọn mức tiền cược để tham gia trận đấu .... ")
	s.gameState = pb.GameState_GameStatePreparing
	fmt.Println("Thiết lập trạng thái match = ", s.gameState)
	s.allowBet = true
	fmt.Println("Thiết lập trạng thái cho phép cược = ", s.allowBet)

	if s.allowBet {
		fmt.Println("Thiết lập mức cược của PlayerIsDealer ... ")

		// xem xem ai đang là dealer
		// server hay playerIsDealer
		if s.playerIsDealer == "" { // server là dealer đặt cược
			s.AddBet(&pb.ShanGameBet{ // add bet for user 1 = 100
				// với AddBet là tổng mức cược đầu vào để chơi game
				UserId: "",
				Chips:  2000,
			})
		} else { // dealer là user đặt cược
			s.AddBet(&pb.ShanGameBet{ // add bet for user 1 = 100
				// với AddBet là tổng mức cược đầu vào để chơi game
				UserId: s.playerIsDealer,
				Chips:  3000,
			})
		}

		fmt.Println("Thiết lập mức cược cho các player ... ")
		s.AddBet(&pb.ShanGameBet{ // add bet for user 1 = 100
			UserId: "User1",
			Chips:  1000,
		})

		s.AddBet(&pb.ShanGameBet{ // add bet for user 2
			UserId: "User2",
			Chips:  9000,
		})

		s.AddBet(&pb.ShanGameBet{ // add bet for user 2
			UserId: "User2",
			Chips:  9000,
		})
		s.ReduceBet(&pb.ShanGameBet{ // change bet for user 1 = 100 - 50 = 50
			UserId: "User1",
			Chips:  500,
		})

		// lấy ra mức cược player đã đặt theo id tương ứng
		fmt.Println("Bet of User 1: ", s.userBets["User1"].First)          // mong doi = 500
		fmt.Println("Bet of User 2: ", s.userBets["User2"].First)          // mong doi = 18000
		fmt.Println("Bet of Dealer 5: ", s.userBets["User5_dealer"].First) // mong doi = 2000
	} else {
		fmt.Println("Allow Bet chưa được cho phép cược!")
		return
	}

	// fmt.Println("???? User sẽ được server - cho tới trận đấu phù hợp ?
	// 				hay là user sau khi chọn mức cược => được join match đó .... ")
	fmt.Println("user request as player .... click deal ")

	// s.deck := NewDeck()
	s.deck.Shuffle()

	fmt.Println("Chia bài .... cho các user đã đặt cược trong ván game .....")
	s.chiaBaiChoPlayerTuongUng()

	fmt.Println("Số lượng userHand : ", len(s.userHands))
	fmt.Println("Số lượng dealerHand(= 0 nếu server is Dealer) : ", len(s.dealerHand.first))
	fmt.Println("Lần lượt user : đưa ra lựa chọn bốc bài tiếp hay không ?")

	// kiểm tra bài của dealer = shan ko ?
	if len(s.userHands) > 0 {
		isCheck_getMoreCard := false
		dealerPoint, dealerHand := s.dealerHand.Eval() // tính điểm cho dealer và lấy ra type của nó
		if dealerPoint > 0 && dealerHand == pb.ShanGameHandType_SHANGAME_HANDTYPE_SHANGAME_HAND_TYPE_SHAN {
			isCheck_getMoreCard = true
		}
		fmt.Println("Dealer được bài shan không ? = ", isCheck_getMoreCard)

		// dealer ko được pok => user,dealer được bốc bài tiếp
		if !isCheck_getMoreCard { // nếu dealer không được bài shan, các player & dealer tiếp tục được bốc tiếp

			fmt.Println("User1 bốc thêm bài...")
			addCardFor_user1, err := s.deck.Deal(1)
			player1Point, player1Hand := s.userHands["User1"].Eval()
			fmt.Println("Player 1 có point = ", player1Point)
			fmt.Printf("Player 1 có type hand = %+v", player1Hand)
			// check điều kiện khi nào thì player hợp lệ để bốc bài tiếp
			if err == nil && player1Point >= 0 && player1Hand != pb.ShanGameHandType_SHANGAME_HANDTYPE_SHANGAME_HAND_TYPE_SHAN && len(s.userHands["User1"].first) <= 3 {
				fmt.Println("Đủ điều kiện để bốc lá bài thêm cho _ User1")
				fmt.Println("Trước khi set giá trị cho user1:", s.userHands["User1"].first)
				s.userHands["User1"].first = append(s.userHands["User1"].first, addCardFor_user1.Cards...)
			}
			fmt.Printf("Bộ bài sau khi rút lần 2 của user 1: \n%+v", s.userHands["User1"].first)

			fmt.Println("Dealer bốc thêm bài...")
			addCardFor_dealer, err := s.deck.Deal(1)
			if err == nil && len(s.dealerHand.first) <= 3 {
				fmt.Println("Đủ điều kiện để bốc lá bài thêm cho _ Dealer")
				fmt.Println("Trước khi set giá trị cho dealer:", s.dealerHand.first)
				s.dealerHand.first = append(s.dealerHand.first, addCardFor_dealer.Cards...)
			}
			fmt.Println("Bộ bài sau khi rút lần 2 của dealer current: \n", s.dealerHand.first)
		}

		fmt.Println("Tiếp theo các player sẽ thực hiện so bài với dealer ...")
		s.CalPointFor_Player_Dealer() // // kiểm tra user còn đủ điều kiện chơi tiếp ko ? - đã kiểm tra trong hàm ( cả user và player )

	} else {
		fmt.Println("Không có bộ bài nào để xét lose or win!")
		return
	}

	fmt.Printf("====END GAME====\n%v", s.CalcGameFinish())

}
