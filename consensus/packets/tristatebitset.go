/*
 * The Clear BSD License
 *
 * Copyright (c) 2019 Insolar Technologies
 *
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without modification, are permitted (subject to the limitations in the disclaimer below) provided that the following conditions are met:
 *
 *  Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
 *  Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.
 *  Neither the name of Insolar Technologies nor the names of its contributors may be used to endorse or promote products derived from this software without specific prior written permission.
 *
 * NO EXPRESS OR IMPLIED LICENSES TO ANY PARTY'S PATENT RIGHTS ARE GRANTED BY THIS LICENSE. THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 *
 */

package packets

import (
	"bytes"
	"encoding/binary"
	"io"
	"math/bits"

	"github.com/pkg/errors"
)

const lastBitMask = 0x01
const lowLengthSize = 6
const firstBitMask = 0x80
const lastTwoBitsMask = 0x3

// TriStateBitSet bitset implementation.
type TriStateBitSet struct {
	CompressedSet bool

	array *bitArray
}

// NewTriStateBitSet creates and returns a tristatebitset.
func NewTriStateBitSet(size int) (*TriStateBitSet, error) {
	bitset := &TriStateBitSet{
		array: newBitArray(size * 2),
	}
	for i := 0; i < size; i++ {
		err := bitset.changeBitState(i, TimedOut)
		if err != nil {
			return nil, err
		}
	}
	return bitset, nil
}

func (dbs *TriStateBitSet) GetCells(mapper BitSetMapper) ([]BitSetCell, error) {
	return dbs.parseCells(mapper)
}

func (dbs *TriStateBitSet) GetTristateArray() ([]TriState, error) {
	cellSize := int(dbs.array.bitsSize / 2)
	cells := make([]TriState, cellSize)
	var err error
	for i := 0; i < cellSize; i++ {
		cells[i], err = parseState(dbs.array, i)
		if err != nil {
			return nil, errors.Wrap(err, "[ GetTristateArray ] failed to parse TriState")
		}
	}
	return cells, nil
}

func (dbs *TriStateBitSet) ApplyChanges(changes []BitSetCell, mapper BitSetMapper) error {
	for _, cell := range changes {
		index, err := mapper.RefToIndex(cell.NodeID)
		if err != nil {
			return errors.Wrap(err, "[ ApplyChanges ] failed to get index from ref")
		}
		err = dbs.changeBucketState(index, cell.State)
		if err != nil {
			return errors.Wrap(err, "[ ApplyChanges ] failed to change bucket state")
		}
	}
	return nil
}

func (dbs *TriStateBitSet) Serialize() ([]byte, error) {
	var firstByte uint8 // compressed and hBitLength bits
	if dbs.CompressedSet {
		firstByte = 0x01
	} else {
		firstByte = 0x00
	}

	data, err := dbs.array.serialize(dbs.CompressedSet)
	if err != nil {
		return nil, errors.Wrap(err, "[ Serialize ] failed to serialize a bitarray")
	}

	length := len(data)
	var result bytes.Buffer
	firstByte = firstByte << 1
	if bits.Len(uint(length)) > lowLengthSize {
		err = dbs.serializeWithHLength(firstByte, length, &result)
		if err != nil {
			return nil, errors.Wrap(err, "[ Serialize ] failed to serialize first bytes")
		}
	} else {
		err = dbs.serializeWithLLength(firstByte, length, &result)
		if err != nil {
			return nil, errors.Wrap(err, "[ Serialize ] failed to serialize first bytes")
		}
	}

	err = binary.Write(&result, defaultByteOrder, data)
	if err != nil {
		return nil, errors.Wrap(err, "[ Serialize ] failed to write binary")
	}

	return result.Bytes(), nil
}

func (dbs *TriStateBitSet) serializeWithHLength(
	firstByte uint8,
	tmpLen int,
	result *bytes.Buffer,
) error {
	var secondByte uint8 // hBitLength
	firstByte++
	firstByte = firstByte << lowLengthSize // move compressed and hBitLength bits to right
	secondByte = uint8(tmpLen)
	err := binary.Write(result, defaultByteOrder, firstByte)
	if err != nil {
		return errors.Wrap(err, "[ serializeWithHLength ] failed to write binary")
	}
	err = binary.Write(result, defaultByteOrder, secondByte)
	if err != nil {
		return errors.Wrap(err, "[ serializeWithHLength ] failed to write binary")
	}
	return nil
}

func (dbs *TriStateBitSet) serializeWithLLength(
	firstByte uint8,
	tmpLen int,
	result *bytes.Buffer,
) error {
	firstByte = firstByte << lowLengthSize // move compressed and hbit flags to right
	firstByte += uint8(tmpLen)
	err := binary.Write(result, defaultByteOrder, firstByte)
	if err != nil {
		return errors.Wrap(err, "[ serializeWithLLength ] failed to write binary")
	}
	return nil
}

func DeserializeBitSet(data io.Reader) (BitSet, error) {
	firstbyte := uint8(0)
	err := binary.Read(data, defaultByteOrder, &firstbyte)
	if firstbyte == 0 {
		return nil, errors.New("[ DeserializeBitSet ] failed to deserialize: wrong data")
	}

	var array *bitArray
	if err != nil {
		return nil, errors.Wrap(err, "[ Deserialize ] failed to read first byte")
	}
	compressed, hbitFlag, length := parseFirstByte(firstbyte)
	if hbitFlag {
		err = binary.Read(data, defaultByteOrder, &length)
		if err != nil {
			return nil, errors.Wrap(err, "[ Deserialize ] failed to read second byte")
		}
	}
	if err != nil {
		return nil, errors.Wrap(err, "[ Deserialize ] failed to create a bitset")
	}
	if compressed {
		array, err = deserializeCompressed(data, int(length))
		if err != nil {
			return nil, errors.Wrap(err, "[ DeserializeBitSet ] failed to deserialize a compressed bitarray")
		}
	} else {
		payload := make([]uint8, length)
		for i := uint8(0); i < length; i++ {
			err := binary.Read(data, defaultByteOrder, &payload[i])
			if err != nil {
				return nil, errors.Wrap(err, "[ Deserialize ] failed to read payload")
			}
		}
		array, err = parseBitArray(payload)
		if err != nil {
			return nil, errors.Wrap(err, "[ Deserialize ] failed to parse a bitarray")
		}
	}
	bitset := &TriStateBitSet{
		array: array,
	}
	return bitset, nil
}

func (dbs *TriStateBitSet) parseCells(mapper BitSetMapper) ([]BitSetCell, error) {
	cellSize := int(dbs.array.bitsSize / 2)
	cells := make([]BitSetCell, cellSize)
	for i := 0; i < cellSize; i++ {
		id, err := mapper.IndexToRef(i)
		if err != nil {
			return nil, err
		}
		cells[i].NodeID = id
		cells[i].State, err = parseState(dbs.array, i)
		if err != nil {
			return nil, errors.Wrap(err, "[ parseCells ] failed to parse TriState")
		}
	}
	return cells, nil
}

func deserializeCompressed(data io.Reader, size int) (*bitArray, error) {
	count := uint16(0)
	value := uint8(0)
	var payload []uint8
	blockSize := 0
	block := uint8(0)
	var err error
	for i := 1; i < size; i = i + 3 { // i := 1 to skip first byte which was read at prev func
		err = binary.Read(data, binary.BigEndian, &count)
		if err != nil {
			return nil, errors.Wrap(err, "[ deserializeCompressed ] failed to read from data")
		}
		err = binary.Read(data, binary.BigEndian, &value)
		if err != nil {
			return nil, errors.Wrap(err, "[ deserializeCompressed ] failed to read from data")
		}
		for j := uint16(0); j < count; j++ {
			block += value
			blockSize += 2
			if blockSize >= sizeOfBlock {
				payload = append(payload, block)
				blockSize = 0
			}
			block = block << 2
		}
	}
	if (blockSize != 0) && (blockSize < sizeOfBlock) {
		block += value
		blockSize += 2
		block = block << uint(sizeOfBlock-blockSize)
		payload = append(payload, block)
	}
	return parseBitArray(payload)
}

func parseState(array *bitArray, index int) (TriState, error) {
	state, err := array.getState(index)
	return TriState(state), err
}

func parseBitArray(payload []uint8) (*bitArray, error) {
	len := len(payload)
	lastByte := payload[len-1]
	array := newBitArray(len*sizeOfBlock - getEmptyBitsCount(lastByte))
	for i := 0; i < len; i++ {
		array.array[i] = payload[i]
	}
	return array, nil
}

func getEmptyBitsCount(byte uint8) int {
	emptyStates := 0
	for i := 0; i < 4; i++ {
		if (byte & lastTwoBitsMask) == 0 {
			emptyStates++
			byte >>= 2
		}
	}
	return emptyStates * 2 // cuz calculated
}

func (dbs *TriStateBitSet) changeBucketState(index int, newState TriState) error {
	return dbs.changeBitState(index, newState)
}

func (dbs *TriStateBitSet) putLastBit(state TriState, index int) error {
	bit := int(state & lastBitMask)
	return dbs.array.put(bit, index)
}

func (dbs *TriStateBitSet) changeBitState(i int, state TriState) error {
	err := dbs.putLastBit(state>>1, 2*i) // put first bit to array
	if err != nil {
		return errors.Wrap(err, "[ changeBitState ] failed to put last bit")
	}
	err = dbs.putLastBit(state, 2*i+1) // put second bit to array
	if err != nil {
		return errors.Wrap(err, "[ changeBitState ] failed to put last bit")
	}
	return nil
}

func parseFirstByte(b uint8) (compressed bool, hbitFlag bool, lbitLength uint8) {
	lbitLength = uint8(0)
	compressed = false
	hbitFlag = false
	if (b & firstBitMask) == firstBitMask { // check compressed flag bit
		compressed = true
	}
	check := (b << 1) & firstBitMask // check hBitLength flag bit
	if check == firstBitMask {
		hbitFlag = true
		return
	}
	lbitLength = (b << 2) >> 2 // remove 2 first bits
	return
}
