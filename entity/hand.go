package entity

import (
	"fmt"
	"strings"

	pb "github.com/nakamaFramework/cgp-common/proto"
)

type Hand struct {
	userId string
	first  []*pb.Card // list card of user current
}

func NewHand(userId string, first []*pb.Card) *Hand {
	return &Hand{
		userId: userId,
		first:  first,
	}
}

func NewHandFromPb(v *pb.ShanGamePlayerHand) *Hand {
	return &Hand{
		userId: v.UserId,
		first:  v.First.Cards,
	}
}

// convert hand to shangameplayerhand dưới dạng proto buffer
func (h *Hand) ToPb() *pb.ShanGamePlayerHand {
	point1, hand1Type := h.Eval() // use func eval to get point , type hand
	return &pb.ShanGamePlayerHand{
		UserId: h.userId,
		First: &pb.ShanGameHand{
			Cards: make([]*pb.Card, 0),
			Point: point1,
			Type:  hand1Type,
		},
	}
}

// lay ra tung point
func getCardPoint(r pb.CardRank) int32 {
	switch v := int32(r); {
	case v <= 9:
		return v
	default:
		return 0
	}
}

// tinh tong diem cho bo bai
func calculatePoint(cards []*pb.Card) int32 {
	if cards == nil { // card : bo bai = nil, tương ứng chưa được khởi tạo và cũng chưa được gán giá trị gì
		return 0
	}
	point := int32(0) // tong diem
	for _, c := range cards {
		v := getCardPoint(c.Rank) // lay point thong qua rank, gan vao v
		point += v                // cong cac gia tri quan bai
	}
	if point >= 10 {
		return (point / 10) % 10
	}

	return point
}

// tinh tong theo so luong that cua la bai
func CalculatePoint_withFactValueCard(cards []*pb.Card) int32 {
	if cards == nil { // card : bo bai = nil, tương ứng chưa được khởi tạo và cũng chưa được gán giá trị gì
		return 0
	}
	point := int32(0) // tong diem
	for _, c := range cards {
		v := getCardPoint(c.Rank) // lay point thong qua rank, gan vao v
		point += v                // cong cac gia tri quan bai
	}

	return point
}

func (h *Hand) isCheckHandType_SHAN(point int32) bool {
	return (point == 9 || point == 8) && len(h.first) == 2
}

func (h *Hand) getCardName_CardSuitOfOneDeck() ([]string, []string) {
	arr_Card_Name := []string{} // gia tri ten loai bai: J, Q, 2,3 ..
	arr_Card_Suit := []string{} // gia tri loai la bai : co, tep, ...

	for i := 0; i < len(h.first); i++ {
		rankName := h.first[i].GetRank().String() // name of card
		// fmt.Println("\n Rank name = ", rankName)
		index := strings.LastIndex(rankName, string('_'))

		arr_Card_Name = append(arr_Card_Name, rankName[index+1:])
		// fmt.Printf("Card name: %+v", arr_Card_Name)

		arr_Card_Suit = append(arr_Card_Suit, h.first[i].Suit.String())
		// fmt.Printf("Card suit: %+v", arr_Card_Suit[i])
	}

	return arr_Card_Name, arr_Card_Suit
}

func (h *Hand) isCheck_sameCardName() bool {
	return (h.first[0].GetRank().String() == h.first[1].GetRank().String()) && (h.first[0].GetRank().String() == h.first[2].GetRank().String())
}

func (h *Hand) isCheckHandType_DIA(arr_Card_Name []string) bool {
	return (arr_Card_Name[0] == "J" || arr_Card_Name[1] == "J" || arr_Card_Name[2] == "J") &&
		(arr_Card_Name[0] == "Q" || arr_Card_Name[1] == "Q" || arr_Card_Name[2] == "Q") &&
		(arr_Card_Name[0] == "K" || arr_Card_Name[1] == "K" || arr_Card_Name[2] == "K")
}

func (h *Hand) isCheck_SameCardSuit(arr_card_suit []string) bool {
	return (arr_card_suit[0] == arr_card_suit[1]) && (arr_card_suit[0] == arr_card_suit[2])
}

func (h *Hand) sordDeck() {
	for i := 0; i < len(h.first)-1; i++ {
		for j := 1; j < len(h.first)-1-i; j++ {
			if *h.first[j].GetRank().Enum() < *h.first[j+1].GetRank().Enum() {
				h.first[j], h.first[j+1] = h.first[j+1], h.first[j]
			}
		}
	}
}

// làm sao để loại bỏ trường hợp: 123, KA2 không được tính
// 123 = 6
// KA2 = 16
func (h *Hand) isCheck_3ConsecutiveCards() bool { // check 3 card liền nhau
	isCheck := 0
	sumCard := calculatePoint(h.first)
	if sumCard == 6 || sumCard == 16 {
		return false
	}

	for i := 0; i < len(h.first); i++ {
		if *h.first[i].GetRank().Enum()+1 == *h.first[i+1].GetRank().Enum() {
			isCheck++
		}
	}
	return isCheck == 2
}

// cùng chất và liên tiếp nhau

func (h *Hand) isCheck_HandType_THUNG_PHA_SANH(arr_card_suit []string) bool {
	h.sordDeck()
	// fmt.Printf("\nBộ bài sau khi sắp xếp: %+v", h.first)
	return h.isCheck_SameCardSuit(arr_card_suit) && h.isCheck_3ConsecutiveCards()
}

// Eval(1) if want to evaluate 1st hand, any else for 2nd hand
func (h *Hand) Eval() (int32, pb.ShanGameHandType) { // return point, type of hand
	point := int32(0)
	point = calculatePoint(h.first)
	// shan
	if h.isCheckHandType_SHAN(point) {
		// fmt.Println("Bộ bài của user: ", h.userId, " thuộc loại Shan", ", với point = ", point)
		return point, pb.ShanGameHandType_SHANGAME_HAND_TYPE_SHAN // shan
	} else if len(h.first) == 3 {
		// Bài xám cô (3 lá giống nhau) // giong nhau ve luong, list cac gia tri cua la bai
		arr_Card_Name, arr_Card_Suit := h.getCardName_CardSuitOfOneDeck()

		// fmt.Println("\nXem thông tin lá bài - (tên lá bài): ", arr_Card_Name[0], ", ", arr_Card_Name[1], ", ", arr_Card_Name[2])
		// fmt.Println("Xem thông tin lá bài - (loại lá bài): ", arr_Card_Suit[0], ", ", arr_Card_Suit[1], ", ", arr_Card_Suit[2])

		// xam co Bài có 3 lá giống nhau , ví dụ: AAA > KKK > QQQ > … > 333 > 222
		if h.isCheck_sameCardName() {

			return point, pb.ShanGameHandType_SHANGAME_HAND_TYPE_XAM_CO

		} else if h.isCheckHandType_DIA(arr_Card_Name) { // 3 con đầu người (J, Q, K)

			return point, pb.ShanGameHandType_SHANGAME_HAND_TYPE_DIA

		} else { // Thùng phá sảnh (3 lá liền nhau & cùng chất)

			if h.isCheck_HandType_THUNG_PHA_SANH(arr_Card_Suit) {
				return point, pb.ShanGameHandType_SHANGAME_HAND_TYPE_THUNG_PHA_SANH // Thùng phá sảnh
			}
		}
	} else {
		// fmt.Println("Bộ bài của user: ", h.userId, " thuộc loại: Normal", ", với point = ", point)
		return point, pb.ShanGameHandType_SHANGAME_HAND_TYPE_NORMAL // normal
	}

	// fmt.Println("Bộ bài của user: ", h.userId, " thuộc loại: Normal", ", với point = ", point)
	return point, pb.ShanGameHandType_SHANGAME_HAND_TYPE_NORMAL
}

// them bai vao bo bai
func (h *Hand) AddCards(c []*pb.Card) {
	h.first = append(h.first, c...)
}

// comparing player hand with dealer hand, -1 -> lost, 1 -> win, 0 -> tie
// return result compare user with dealer
func (h *Hand) Compare(d *Hand) int { // h: player, d: dealer
	// fmt.Println("\n\n=====Thực hiện so sánh: player_", h.userId, " với dealer_ ", d.userId)
	player_point, player_handType := h.Eval()
	dealer_point, dealer_handType := d.Eval()
	// fmt.Println("\nPoint of player - ", h.userId, " = ", player_point, " , Type of Hand - ", player_handType)
	// fmt.Println("Point of player - ", d.userId, " = ", dealer_point, " , Type of Hand - ", dealer_handType)

	if int(player_handType) > int(dealer_handType) { // so sanh type , type trong nay cung chua gia tri duoc dinh nghia trong file blackjack_api.pb.go
		return 1
	} else if int(player_handType) == int(dealer_handType) { // neu gia tri type = nhau
		// if type = xam => ko so sanh point nhu thong thuong duoc
		if player_handType == pb.ShanGameHandType_SHANGAME_HAND_TYPE_XAM_CO {
			result_Xam := CompareHandType_XAM_CO(h.first, d.first)
			return result_Xam
		} else if player_handType == pb.ShanGameHandType_SHANGAME_HAND_TYPE_DIA {
			// Bài cùng bộ đầu người là hòa
			if CalculatePoint_withFactValueCard(h.first) == CalculatePoint_withFactValueCard(d.first) {
				return 0
			}
		} else if player_handType == pb.ShanGameHandType_SHANGAME_HAND_TYPE_THUNG_PHA_SANH {
			// thung pha sanh
			// so sanh theo thu tu = gia tri
			result := CompareHandType_THUNG_PHA_SANH_byRank(h.first, d.first)

			if result > 0 {
				return result
			}
			// result = 0
			// so sanh theo chat ( neu = gia tri )
			return h.compareBySuit(d.first)
		} else {
			// normal
			if (len(h.first) < len(d.first)) || (player_point > dealer_point) { // so sanh diem
				return 1
			} else if player_point < dealer_point {
				return -1
			} else { // p_point = d_point && p_type = d_type
				if player_handType == pb.ShanGameHandType_SHANGAME_HAND_TYPE_SHAN {
					// thuc hien ham so sanh tai day
					return h.compareBySuit(d.first)
				}
			}
		}
	} else {
		return -1
	}
	return 0
}

// get tỉ lệ thắng thua của bộ bài player
func (h *Hand) GetTiLeThangThuaPlayer() int {
	player_point, player_handType := h.Eval()

	// fmt.Println("point of player ", player_point, "List card of player", h.first)
	if player_handType == pb.ShanGameHandType_SHANGAME_HAND_TYPE_SHAN && player_point >= 0 {
		// Tỉ lệ ăn: 1 hoặc  2. Cụ thể: Nhân 2 khi có 2 con cùng chất hoặc là 1 đôi (đôi 4 hoặc đôi 9)
		isCheck := false
		// cùng chất
		// cùng đôi
		if h.first[0].Suit.String() == h.first[1].Suit.String() && h.first[0].Rank.Number() == h.first[1].Rank.Number() {
			return 2
		}

		if !isCheck {
			return 1
		}
	} else if player_handType == pb.ShanGameHandType_SHANGAME_HAND_TYPE_XAM_CO {
		return 5
	} else if player_handType == pb.ShanGameHandType_SHANGAME_HAND_TYPE_DIA {
		// Tỉ lệ ăn: 3 hoặc 5.
		// Cụ thể: 3 lá đầu người cùng chất (JQK cơ, bích, rô, tép) x5;
		// 3 lá đầu người khác chất x3
		if h.first[0].Suit.String() == h.first[1].Suit.String() && h.first[0].Suit.String() == h.first[2].Suit.String() {
			return 5
		}
		return 3
	} else if player_handType == pb.ShanGameHandType_SHANGAME_HAND_TYPE_THUNG_PHA_SANH {
		return 5
	} else {
		// -> "x2" khi bài có đúng 2 quân và là 2 quân cùng chất hoặc là 1 đôi (Không nhận lá bài thứ 3).
		// Ví dụ: 2 cơ 4 cơ, 5 rô J rô; đôi 2; đôi 3; ...
		// -> "x3" khi bài 3 quân cùng chất. Ví dụ: 3 tép 4 tép Q tép, 7 cơ K cơ Q cơ
		if len(h.first) == 2 &&
			(h.first[0].Suit.String() == h.first[1].Suit.String() || h.first[0].Rank.Number() == h.first[1].Rank.Number()) {

			return 2
		} else if len(h.first) == 3 && h.first[0].Suit.String() == h.first[1].Suit.String() && h.first[0].Suit.String() == h.first[2].Suit.String() {
			return 3
		}
		return 1
	}

	return 1
}

func (h *Hand) JoinCardsToString(listCard []*pb.Card) string {
	cardsString := ""
	for i := 0; i < len(listCard); i++ {
		rankName := listCard[i].GetRank().String() // name of card
		// index := strings.LastIndex(rankName, string('_'))
		// cardsString += rankName[index+1:]
		cardsString += rankName
	}

	return cardsString
}

// get la bai cao nhat theo ranking
func (h *Hand) GetMaxCardByRanking_ShanType(listCard []*pb.Card) (int, int) {
	max_ranking := 2
	max_value_suit := 0
	// theo ranking tăng dần: 2,3,4,5,6...10,J,Q,K,A
	for _, card := range listCard {
		cardValue := int(card.Rank.Number())
		fmt.Println("Card value current :", cardValue)
		cardSuit := int(card.Suit.Number())
		fmt.Println("Card suit current :", cardSuit)
		if cardValue == 1 {
			return 1, cardSuit
		}

		if max_ranking < cardValue {
			max_value_suit = cardSuit
			return cardValue, cardSuit
		}
	}

	return max_ranking, max_value_suit
}

func CompareHandType_XAM_CO(playerCard, dealerCard []*pb.Card) int {
	// lấy ra value của 3 lá bài
	sum_playerCard := CalculatePoint_withFactValueCard(playerCard)
	sum_dealerCard := CalculatePoint_withFactValueCard(dealerCard)
	if sum_playerCard == 3 {
		return 1
	}
	if int(sum_playerCard) > int(sum_dealerCard) {
		return 1
	}

	return -1

	// with type hand : XAM_CO khong co truong hop = nhau

}

func CompareHandType_THUNG_PHA_SANH_byRank(hand1, hand2 []*pb.Card) int {
	sum_hand1 := CalculatePoint_withFactValueCard(hand1)
	sum_hand2 := CalculatePoint_withFactValueCard(hand2)

	// Compare based on predefined ranking
	if sum_hand1 > sum_hand2 {
		return 1 // hand1 is higher
	} else if sum_hand1 < sum_hand2 {
		return -1
	} else { // bang rank - so theo chat
		return 0
	}

}

// so sanh bai dang shan
func (h *Hand) compareBySuit(d []*pb.Card) int {
	// truyen vao ds la bai cua user
	player_maxCard, player_maxCard_suit := h.GetMaxCardByRanking_ShanType(h.first)
	dealer_maxCard, dealer_maxCard_suit := h.GetMaxCardByRanking_ShanType(d)

	if player_maxCard > dealer_maxCard {
		return 1
	} else if player_maxCard < dealer_maxCard {
		return -1
	} else {
		// bằng điểm, cùng số lá bài, cùng cây cao nhất
		// so sánh chất của lá bài theo ranking: Bích > Rô > Cơ > Tép
		if player_maxCard_suit < dealer_maxCard_suit { // do value suit: heart =1 , SPADES  = 4
			return 1
		}
		return -1
	}
}

// Dealer must draw on lower than 17 and stand on >= 17
func (h *Hand) DealerMustDraw() bool {
	return calculatePoint(h.first) < 17
}

func (h *Hand) DealerPotentialBlackjack() bool {
	return h.first[0].Rank == pb.CardRank_RANK_A
}

// Check if player can draw on current hand, call with pos=1 for 1st hand, else 2nd hand
func (h *Hand) PlayerCanDraw(pos pb.ShanGameHandN0) bool {
	// 	khi nào user có thể bốc thêm bài ?
	// tổng lá bài = 2 thì đc bốc thêm 1 lá
	// không phải bài shan ( khi tổng điểm != 8 or !=9 )
	point := calculatePoint(h.first)
	if len(h.first) == 2 && (point != 8 && point != 9) {
		return true
	}
	return false
}

// func (h *Hand) PlayerCanSplit() bool {
// 	return (h.second == nil || len(h.second) == 0) &&
// 		len(h.first) == 2 &&
// 		getCardPoint(h.first[0].Rank) == getCardPoint(h.first[1].Rank)
// }

// func (h *Hand) Split() {
// 	h.second = []*pb.Card{
// 		h.first[1], // lấy lá bài tại vị trí 1, gán vào hand 2
// 	}
// 	h.first = []*pb.Card{
// 		h.first[0], // lấy lá bài tại vị trí 0, gán vào hand 1
// 	}
// }
