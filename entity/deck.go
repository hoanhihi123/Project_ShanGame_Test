package entity

import (
	"errors"
	"fmt"
	"math/rand"

	pb "github.com/nakamaFramework/cgp-common/proto"
)

// max deal can deal
const MaxCard = 52

type Deck struct {
	ListCard *pb.ListCard
	Dealt    int
}

// khoi tao bo bai moi, chua gan gia tri
func NewDeck() *Deck {
	ranks := []pb.CardRank{
		pb.CardRank_RANK_A,
		pb.CardRank_RANK_2,
		pb.CardRank_RANK_3,
		pb.CardRank_RANK_4,
		pb.CardRank_RANK_5,
		pb.CardRank_RANK_6,
		pb.CardRank_RANK_7,
		pb.CardRank_RANK_8,
		pb.CardRank_RANK_9,
		pb.CardRank_RANK_10,
		pb.CardRank_RANK_J,
		pb.CardRank_RANK_Q,
		pb.CardRank_RANK_K,
	}

	suits := []pb.CardSuit{
		pb.CardSuit_SUIT_CLUBS,
		pb.CardSuit_SUIT_DIAMONDS,
		pb.CardSuit_SUIT_HEARTS,
		pb.CardSuit_SUIT_SPADES,
	}

	cards := &pb.ListCard{}
	for i := 0; i < 1; i++ {
		for _, r := range ranks {
			for _, s := range suits {
				cards.Cards = append(cards.Cards, &pb.Card{
					Rank: r,
					Suit: s,
				})
			}
		}
	}
	return &Deck{
		Dealt:    0,
		ListCard: cards,
	}
}

// tron bai
func (d *Deck) Shuffle() {
	for i := 1; i < len(d.ListCard.Cards); i++ {
		r := rand.Intn(i + 1)
		if i != r {
			d.ListCard.Cards[r], d.ListCard.Cards[i] = d.ListCard.Cards[i], d.ListCard.Cards[r]
		}
	}
}

// chia bai
func (d *Deck) Deal(n int) (*pb.ListCard, error) {
	if (MaxCard - d.Dealt) < n {
		return nil, errors.New("deck.deal.error-not-enough")
	}
	var cards pb.ListCard
	for i := 0; i < n; i++ { // n số lá muốn chia
		cards.Cards = append(cards.Cards, d.ListCard.Cards[d.Dealt]) // lấy bài từ listCard và cho vào cards
		d.Dealt++                                                    // số lá bài đã chia
	}
	fmt.Println("D.dealt current = ", d.Dealt)
	return &cards, nil // trả về card sau khi chia
}
