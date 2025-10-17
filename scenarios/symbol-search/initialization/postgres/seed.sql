-- Symbol Search Seed Data
-- Essential Unicode categories, blocks, and sample characters

-- Insert Unicode Categories
INSERT INTO categories (code, name, description) VALUES
('Lu', 'Letter, Uppercase', 'An uppercase letter'),
('Ll', 'Letter, Lowercase', 'A lowercase letter'),
('Lt', 'Letter, Titlecase', 'A digraphic character, with first part uppercase'),
('Lm', 'Letter, Modifier', 'A modifier letter'),
('Lo', 'Letter, Other', 'Other letters, including syllables and ideographs'),
('Mn', 'Mark, Nonspacing', 'A nonspacing combining mark (zero advance width)'),
('Mc', 'Mark, Spacing Combining', 'A spacing combining mark (positive advance width)'),
('Me', 'Mark, Enclosing', 'An enclosing combining mark'),
('Nd', 'Number, Decimal Digit', 'A decimal digit'),
('Nl', 'Number, Letter', 'A letterlike numeric character'),
('No', 'Number, Other', 'A numeric character of other type'),
('Pc', 'Punctuation, Connector', 'A connecting punctuation mark, like a tie'),
('Pd', 'Punctuation, Dash', 'A dash or hyphen punctuation mark'),
('Ps', 'Punctuation, Open', 'An opening punctuation mark (of a pair)'),
('Pe', 'Punctuation, Close', 'A closing punctuation mark (of a pair)'),
('Pi', 'Punctuation, Initial quote', 'An initial quotation mark'),
('Pf', 'Punctuation, Final quote', 'A final quotation mark'),
('Po', 'Punctuation, Other', 'A punctuation mark of other type'),
('Sm', 'Symbol, Math', 'A symbol of mathematical use'),
('Sc', 'Symbol, Currency', 'A currency sign'),
('Sk', 'Symbol, Modifier', 'A non-letterlike modifier symbol'),
('So', 'Symbol, Other', 'A symbol of other type'),
('Zs', 'Separator, Space', 'A space character (of non-zero width)'),
('Zl', 'Separator, Line', 'U+2028 LINE SEPARATOR only'),
('Zp', 'Separator, Paragraph', 'U+2029 PARAGRAPH SEPARATOR only'),
('Cc', 'Other, Control', 'A C0 or C1 control code'),
('Cf', 'Other, Format', 'A format control character'),
('Cs', 'Other, Surrogate', 'A surrogate code point'),
('Co', 'Other, Private Use', 'A private-use character'),
('Cn', 'Other, Not Assigned', 'A reserved unassigned code point or a noncharacter')
ON CONFLICT (code) DO NOTHING;

-- Insert Unicode Blocks (Essential ones)
INSERT INTO character_blocks (name, start_codepoint, end_codepoint, description) VALUES
('Basic Latin', 0, 127, 'ASCII characters'),
('Latin-1 Supplement', 128, 255, 'ISO Latin-1 characters'),
('Latin Extended-A', 256, 383, 'European Latin characters'),
('Latin Extended-B', 384, 591, 'Additional European and African Latin characters'),
('IPA Extensions', 592, 687, 'International Phonetic Alphabet extensions'),
('Spacing Modifier Letters', 688, 767, 'Modifier letters'),
('Combining Diacritical Marks', 768, 879, 'Combining diacritical marks'),
('Greek and Coptic', 880, 1023, 'Greek and Coptic characters'),
('Cyrillic', 1024, 1279, 'Cyrillic characters'),
('Cyrillic Supplement', 1280, 1327, 'Additional Cyrillic characters'),
('Armenian', 1328, 1423, 'Armenian characters'),
('Hebrew', 1424, 1535, 'Hebrew characters'),
('Arabic', 1536, 1791, 'Arabic characters'),
('Syriac', 1792, 1871, 'Syriac characters'),
('Arabic Supplement', 1872, 1919, 'Additional Arabic characters'),
('Thaana', 1920, 1983, 'Thaana characters'),
('General Punctuation', 8192, 8303, 'General punctuation'),
('Superscripts and Subscripts', 8304, 8351, 'Superscripts and subscripts'),
('Currency Symbols', 8352, 8399, 'Currency symbols'),
('Letterlike Symbols', 8448, 8527, 'Letterlike symbols'),
('Number Forms', 8528, 8591, 'Number forms'),
('Arrows', 8592, 8703, 'Arrow symbols'),
('Mathematical Operators', 8704, 8959, 'Mathematical operators'),
('Miscellaneous Technical', 8960, 9215, 'Miscellaneous technical symbols'),
('Control Pictures', 9216, 9279, 'Control pictures'),
('Optical Character Recognition', 9280, 9311, 'OCR symbols'),
('Enclosed Alphanumerics', 9312, 9471, 'Enclosed alphanumerics'),
('Box Drawing', 9472, 9599, 'Box drawing characters'),
('Block Elements', 9600, 9631, 'Block elements'),
('Geometric Shapes', 9632, 9727, 'Geometric shapes'),
('Miscellaneous Symbols', 9728, 9983, 'Miscellaneous symbols'),
('Dingbats', 9984, 10175, 'Dingbat symbols'),
('Emoticons', 128512, 128591, 'Emoji emoticons'),
('Miscellaneous Symbols and Pictographs', 127744, 128511, 'Additional symbols and pictographs'),
('Transport and Map Symbols', 128640, 128767, 'Transport and map symbols'),
('Alchemical Symbols', 128768, 128895, 'Alchemical symbols'),
('CJK Unified Ideographs', 19968, 40959, 'CJK unified ideographs')
ON CONFLICT (name) DO NOTHING;

-- Insert sample characters (Essential ASCII + common symbols + emojis)
INSERT INTO characters (codepoint, decimal, name, category, block, unicode_version, description, html_entity, css_content, properties) VALUES

-- Basic ASCII (sample)
('U+0020', 32, 'SPACE', 'Zs', 'Basic Latin', '1.1', 'Space character', '&#32;', '\0020', '{"general_category": "Zs"}'),
('U+0021', 33, 'EXCLAMATION MARK', 'Po', 'Basic Latin', '1.1', 'Exclamation mark', '&#33;', '\0021', '{"general_category": "Po"}'),
('U+0023', 35, 'NUMBER SIGN', 'Po', 'Basic Latin', '1.1', 'Hash, pound sign', '&#35;', '\0023', '{"general_category": "Po"}'),
('U+0024', 36, 'DOLLAR SIGN', 'Sc', 'Basic Latin', '1.1', 'Dollar sign', '&#36;', '\0024', '{"general_category": "Sc"}'),
('U+0025', 37, 'PERCENT SIGN', 'Po', 'Basic Latin', '1.1', 'Percent sign', '&#37;', '\0025', '{"general_category": "Po"}'),
('U+0026', 38, 'AMPERSAND', 'Po', 'Basic Latin', '1.1', 'Ampersand', '&amp;', '\0026', '{"general_category": "Po"}'),
('U+002A', 42, 'ASTERISK', 'Po', 'Basic Latin', '1.1', 'Asterisk', '&#42;', '\002A', '{"general_category": "Po"}'),
('U+002B', 43, 'PLUS SIGN', 'Sm', 'Basic Latin', '1.1', 'Plus sign', '&#43;', '\002B', '{"general_category": "Sm"}'),
('U+002D', 45, 'HYPHEN-MINUS', 'Pd', 'Basic Latin', '1.1', 'Hyphen or minus sign', '&#45;', '\002D', '{"general_category": "Pd"}'),
('U+002F', 47, 'SOLIDUS', 'Po', 'Basic Latin', '1.1', 'Forward slash', '&#47;', '\002F', '{"general_category": "Po"}'),

-- Numbers
('U+0030', 48, 'DIGIT ZERO', 'Nd', 'Basic Latin', '1.1', 'Digit zero', '&#48;', '\0030', '{"general_category": "Nd"}'),
('U+0031', 49, 'DIGIT ONE', 'Nd', 'Basic Latin', '1.1', 'Digit one', '&#49;', '\0031', '{"general_category": "Nd"}'),
('U+0032', 50, 'DIGIT TWO', 'Nd', 'Basic Latin', '1.1', 'Digit two', '&#50;', '\0032', '{"general_category": "Nd"}'),

-- Letters (sample)
('U+0041', 65, 'LATIN CAPITAL LETTER A', 'Lu', 'Basic Latin', '1.1', 'Capital letter A', '&#65;', '\0041', '{"general_category": "Lu"}'),
('U+0061', 97, 'LATIN SMALL LETTER A', 'Ll', 'Basic Latin', '1.1', 'Small letter a', '&#97;', '\0061', '{"general_category": "Ll"}'),

-- Common symbols
('U+00A9', 169, 'COPYRIGHT SIGN', 'So', 'Latin-1 Supplement', '1.1', 'Copyright symbol', '&copy;', '\00A9', '{"general_category": "So"}'),
('U+00AE', 174, 'REGISTERED SIGN', 'So', 'Latin-1 Supplement', '1.1', 'Registered trademark symbol', '&reg;', '\00AE', '{"general_category": "So"}'),

-- Mathematical operators
('U+2200', 8704, 'FOR ALL', 'Sm', 'Mathematical Operators', '1.1', 'Universal quantifier', '&#8704;', '\2200', '{"general_category": "Sm"}'),
('U+2203', 8707, 'THERE EXISTS', 'Sm', 'Mathematical Operators', '1.1', 'Existential quantifier', '&#8707;', '\2203', '{"general_category": "Sm"}'),
('U+2208', 8712, 'ELEMENT OF', 'Sm', 'Mathematical Operators', '1.1', 'Set membership', '&#8712;', '\2208', '{"general_category": "Sm"}'),
('U+221E', 8734, 'INFINITY', 'Sm', 'Mathematical Operators', '1.1', 'Infinity symbol', '&#8734;', '\221E', '{"general_category": "Sm"}'),

-- Arrows  
('U+2190', 8592, 'LEFTWARDS ARROW', 'Sm', 'Arrows', '1.1', 'Left arrow', '&#8592;', '\2190', '{"general_category": "Sm"}'),
('U+2191', 8593, 'UPWARDS ARROW', 'Sm', 'Arrows', '1.1', 'Up arrow', '&#8593;', '\2191', '{"general_category": "Sm"}'),
('U+2192', 8594, 'RIGHTWARDS ARROW', 'Sm', 'Arrows', '1.1', 'Right arrow', '&#8594;', '\2192', '{"general_category": "Sm"}'),
('U+2193', 8595, 'DOWNWARDS ARROW', 'Sm', 'Arrows', '1.1', 'Down arrow', '&#8595;', '\2193', '{"general_category": "Sm"}'),

-- Heart symbols
('U+2764', 10084, 'HEAVY BLACK HEART', 'So', 'Miscellaneous Symbols', '1.1', 'Black heart symbol', '&#10084;', '\2764', '{"general_category": "So"}'),
('U+2665', 9829, 'BLACK HEART SUIT', 'So', 'Miscellaneous Symbols', '1.1', 'Playing card heart', '&#9829;', '\2665', '{"general_category": "So"}'),

-- Popular emojis  
('U+1F600', 128512, 'GRINNING FACE', 'So', 'Emoticons', '6.1', 'Grinning face emoji', '&#128512;', '\1F600', '{"general_category": "So", "emoji": true}'),
('U+1F601', 128513, 'GRINNING FACE WITH SMILING EYES', 'So', 'Emoticons', '6.0', 'Grinning face with smiling eyes', '&#128513;', '\1F601', '{"general_category": "So", "emoji": true}'),
('U+1F602', 128514, 'FACE WITH TEARS OF JOY', 'So', 'Emoticons', '6.0', 'Face with tears of joy', '&#128514;', '\1F602', '{"general_category": "So", "emoji": true}'),
('U+1F603', 128515, 'SMILING FACE WITH OPEN MOUTH', 'So', 'Emoticons', '6.0', 'Smiling face with open mouth', '&#128515;', '\1F603', '{"general_category": "So", "emoji": true}'),
('U+1F604', 128516, 'SMILING FACE WITH OPEN MOUTH AND SMILING EYES', 'So', 'Emoticons', '6.0', 'Smiling face with open mouth and smiling eyes', '&#128516;', '\1F604', '{"general_category": "So", "emoji": true}'),

-- More popular symbols
('U+2605', 9733, 'BLACK STAR', 'So', 'Miscellaneous Symbols', '1.1', 'Solid star', '&#9733;', '\2605', '{"general_category": "So"}'),
('U+2606', 9734, 'WHITE STAR', 'So', 'Miscellaneous Symbols', '1.1', 'Outline star', '&#9734;', '\2606', '{"general_category": "So"}'),
('U+2713', 10003, 'CHECK MARK', 'So', 'Miscellaneous Symbols', '1.1', 'Check mark', '&#10003;', '\2713', '{"general_category": "So"}'),
('U+2717', 10007, 'BALLOT X', 'So', 'Miscellaneous Symbols', '1.1', 'X mark', '&#10007;', '\2717', '{"general_category": "So"}'),

-- Currency symbols
('U+20AC', 8364, 'EURO SIGN', 'Sc', 'Currency Symbols', '2.1', 'Euro currency symbol', '&#8364;', '\20AC', '{"general_category": "Sc"}'),
('U+00A3', 163, 'POUND SIGN', 'Sc', 'Latin-1 Supplement', '1.1', 'British pound symbol', '&#163;', '\00A3', '{"general_category": "Sc"}'),
('U+00A5', 165, 'YEN SIGN', 'Sc', 'Latin-1 Supplement', '1.1', 'Japanese yen symbol', '&#165;', '\00A5', '{"general_category": "Sc"}'),

-- Box drawing characters (useful for terminal UIs)
('U+2500', 9472, 'BOX DRAWINGS LIGHT HORIZONTAL', 'So', 'Box Drawing', '1.1', 'Horizontal line', '&#9472;', '\2500', '{"general_category": "So"}'),
('U+2502', 9474, 'BOX DRAWINGS LIGHT VERTICAL', 'So', 'Box Drawing', '1.1', 'Vertical line', '&#9474;', '\2502', '{"general_category": "So"}'),
('U+250C', 9484, 'BOX DRAWINGS LIGHT DOWN AND RIGHT', 'So', 'Box Drawing', '1.1', 'Top-left corner', '&#9484;', '\250C', '{"general_category": "So"}'),
('U+2510', 9488, 'BOX DRAWINGS LIGHT DOWN AND LEFT', 'So', 'Box Drawing', '1.1', 'Top-right corner', '&#9488;', '\2510', '{"general_category": "So"}'),
('U+2514', 9492, 'BOX DRAWINGS LIGHT UP AND RIGHT', 'So', 'Box Drawing', '1.1', 'Bottom-left corner', '&#9492;', '\2514', '{"general_category": "So"}'),
('U+2518', 9496, 'BOX DRAWINGS LIGHT UP AND LEFT', 'So', 'Box Drawing', '1.1', 'Bottom-right corner', '&#9496;', '\2518', '{"general_category": "So"}')

ON CONFLICT (codepoint) DO NOTHING;

-- Update statistics for query optimization
ANALYZE categories;
ANALYZE character_blocks; 
ANALYZE characters;