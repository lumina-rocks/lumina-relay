import asyncio
import os
import websockets
import json
from datetime import datetime
from uuid import uuid4

async def relay_websockets(input_websocket, output_websocket, kinds, sub_id):
    while True:
        try:
            # Wait for an event on input websocket
            event = json.loads(await input_websocket.recv())
            try:
                if(event[0] == "EVENT"):
                    print("Received ID: ",event[2]['id']," // Kind: ",event[2]['kind'])
                    # Forward the event to output websocket
                    await output_websocket.send(json.dumps(["EVENT", sub_id, event[2]]))
                elif(event[0] == "EOSE"):
                    print("End of stream")

            except Exception as error:
                print(f"Failed to relay event: {error}")
                if("sent 1011" in str(error)):
                    print("Got Code 1011 -> Closing websockets...")
                    input_websocket.close()
                    output_websocket.close()
                continue

        except websockets.ConnectionClosed:
            # If either websocket is closed, attempt to reconnect
            print("Connection closed, attempting to reconnect...")
            await asyncio.sleep(1)
            break

async def main():
    print("Scraper started...")
    # Read the websocket URLs from environment variables
    input_url = os.environ.get("INPUT_RELAY")
    output_url = os.environ.get("OUTPUT_RELAY")
    kinds = os.environ.get("KINDS")

    # If either relay URL is missing, raise an error
    if not input_url:
        raise ValueError("Please set the INPUT_RELAY environment variable")
    if not output_url:
        raise ValueError("Please set the OUTPUT_RELAY environment variable")

    while True:
        try:
            sub_id = str(uuid4())
            async with websockets.connect(input_url) as input_websocket, \
                       websockets.connect(output_url) as output_websocket:
                message = f'["REQ", "{sub_id}", {{"kinds": {kinds}}}]'
                await input_websocket.send(message)
                await relay_websockets(input_websocket, output_websocket, kinds, sub_id)

        except Exception as error:
            # If the initial connection attempt fails, attempt to reconnect immediately
            print(f"Failed to connect: {error}")
            await asyncio.sleep(1)
            if "maximum recursion depth exceeded" in str(error):
                raise RuntimeError("Maximum recursion depth exceeded, crashing application.")
            continue

# Start the script
asyncio.run(main())