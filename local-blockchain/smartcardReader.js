const { Devices } = require('smartcard');

const devices = new Devices();

devices.on('card-inserted', async ({ reader, card }) => {
    console.log(`Card detected in ${reader.name}`);
    console.log(`ATR: ${card.atr.toString('hex')}`);

    // Get card UID (works with most contactless cards)
    const response = await card.transmit([0xFF, 0xCA, 0x00, 0x00, 0x00]);
    console.log(`UID: ${response.slice(0, -2).toString('hex')}`);
});

devices.on('error', (err) => console.error(err.message));

devices.start();