# soup2esc

Soup2esc takes a HTML-like-input (a tag soup) and produces an
ESC/POS-compatible output. It makes it easy to send stuff to a "receipt"
printer, in other words.

Main things it does:

- Handles all the basic formatting commands
- Also handles properly the justification commands
- Supports nested tags (centered text with formatting per-word/phrase)
- Can do barcodes galore, with options (at least the stuff my printer handles)
- And, for a kicker, it can print a PNG as a raster image (with options how to!)
- Exits on earliest problems

What it **doesn't do**:

- Try to be smart and handle nonsense tag combinations, or going over the margin
- Or stop you from printing out /dev/random
- Being a stream processor it will happily output stuff before it dies

## Showcase

None for now, given nobody will use this anyway :P

But this is a part of a larger project, with the second element being the
[snippetd](https://github.com/drbig/snippetd).
Been using this tandem for less than a month but so far I'm getting what
I wanted, duh.

## Contributing

Follow the usual GitHub development model:

1. Clone the repository
2. Make your changes on a separate branch
3. Make sure you run `gofmt` and `go test` before committing
4. Make a pull request

See licensing for legalese.

## Licensing

Standard two-clause BSD license, see LICENSE.txt for details.

Any contributions will be licensed under the same conditions.

Copyright (c) 2019 Piotr S. Staszewski
