import json
import random
import secrets

from Crypto.Hash import keccak

file_path = './mock.jsonl'

# Starting blockNumber
block_number = 0

# Maximum block number to reach
max_block_number = 100

def random_eth_hash():
    random_bytes = secrets.token_bytes(32)
    keccak_hash = keccak.new(digest_bits=256)
    keccak_hash.update(random_bytes)
    return keccak_hash.hexdigest()

def generate_big_integer_hex():
    # return hex(random.randint(1_000_000_000_000_000_000, 1_000_000_000_000_000_000_000))
    return hex(1)


with open(file_path, 'w') as file:
    parent_hash = "0x0000000000000000000000000000000000000000000000000000000000000000"

    for block_number in range(0, max_block_number):
        hash = '0x' + random_eth_hash()

        data = {
            "blockNumber": block_number,
            "hash": hash,
            "parentHash": parent_hash
        }

        # genesis block
        if block_number == 0:
            data["issuance"] = {
                "genesisAlloc": generate_big_integer_hex(),
            }
        else:
            data["issuance"] = {
                "reward": generate_big_integer_hex(),
                "withdrawals": generate_big_integer_hex(),
            }
            data["burn"] = {
                "eip1559": generate_big_integer_hex(),
                "blob": generate_big_integer_hex(),
                "misc": generate_big_integer_hex(),
            }

        file.write(json.dumps(data) + '\n')

        # Update parent hash for the next block
        parent_hash = hash
