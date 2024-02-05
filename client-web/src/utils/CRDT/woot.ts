export interface CRDT {
  Insert(position: number, value: string): Promise<string>;
  Delete(position: number): Promise<string>;
}

export class Character {
  ID: string;
  Visible: boolean;
  Value: string;
  CP: string;
  CN: string;

  constructor(
    ID: string,
    Visible: boolean,
    Value: string,
    CP: string,
    CN: string
  ) {
    this.ID = ID;
    this.Visible = Visible;
    this.Value = Value;
    this.CP = CP;
    this.CN = CN;
  }
}

export class Doc {
  Characters: Character[];

  static SiteID: number = 0;
  static LocalClock: number = 0;

  static CharacterStart: Character = new Character(
    "start",
    false,
    "",
    "",
    "end"
  );
  static CharacterEnd: Character = new Character("end", false, "", "start", "");

  static ErrPositionOutOfBounds: Error = new Error("position out of bounds");
  static ErrEmptyWCharacter: Error = new Error("empty char ID provided");
  static ErrBoundsNotPresent: Error = new Error(
    "subsequence bound(s) not present"
  );

  constructor() {
    this.Characters = [Doc.CharacterStart, Doc.CharacterEnd];
  }

  // static async Load(fileName: string): Promise<Doc> {
  //   // Implement the load functionality as needed
  //   throw new Error("Load method not implemented");
  // }

  // static async Save(fileName: string, doc: Doc): Promise<void> {
  //   // Implement the save functionality as needed
  //   throw new Error("Save method not implemented");
  // }

  SetText(newDoc: Doc): void {
    for (const char of newDoc.Characters) {
      const c = new Character(
        char.ID,
        char.Visible,
        char.Value,
        char.CP,
        char.CN
      );
      this.Characters.push(c);
    }
  }

  Content(): string {
    let value = "";
    for (const char of this.Characters) {
      if (char.Visible) {
        value += char.Value;
      }
    }
    return value;
  }

  static ContentFromChararacters(chars: Character[]): string {
    let value = "";
    for (const char of chars) {
      if (char.Visible) {
        value += char.Value;
      }
    }
    return value;
  }

  IthVisible(position: number): Character {
    let count = 0;

    for (const char of this.Characters) {
      if (char.Visible) {
        if (count === position - 1) {
          return char;
        }
        count++;
      }
    }

    return new Character("-1", false, "", "", "");
  }

  Length(): number {
    return this.Characters.length;
  }

  ElementAt(position: number): Character {
    if (position < 0 || position >= this.Length()) {
      throw Doc.ErrPositionOutOfBounds;
    }

    return this.Characters[position];
  }

  Position(charID: string): number {
    for (let position = 0; position < this.Characters.length; position++) {
      if (charID === this.Characters[position].ID) {
        return position + 1;
      }
    }

    return -1;
  }

  Left(charID: string): string {
    const i = this.Position(charID);
    if (i <= 0) {
      return this.Characters[i].ID;
    }
    return this.Characters[i - 1].ID;
  }

  Right(charID: string): string {
    const i = this.Position(charID);
    if (i >= this.Characters.length - 1) {
      return this.Characters[i - 1].ID;
    }
    return this.Characters[i + 1].ID;
  }

  Contains(charID: string): boolean {
    const position = this.Position(charID);
    return position !== -1;
  }

  Find(id: string): Character {
    for (const char of this.Characters) {
      if (char.ID === id) {
        return char;
      }
    }

    return new Character("-1", false, "", "", "");
  }

  Subseq(wcharacterStart: Character, wcharacterEnd: Character): Character[] {
    const startPosition = this.Position(wcharacterStart.ID);
    const endPosition = this.Position(wcharacterEnd.ID);

    if (startPosition === -1 || endPosition === -1) {
      throw Doc.ErrBoundsNotPresent;
    }

    if (startPosition > endPosition) {
      throw Doc.ErrBoundsNotPresent;
    }

    if (startPosition === endPosition) {
      return [];
    }

    return this.Characters.slice(startPosition, endPosition - 1);
  }

  LocalInsert(char: Character, position: number): Doc {
    if (position <= 0 || position >= this.Length()) {
      throw Doc.ErrPositionOutOfBounds;
    }

    if (char.ID === "") {
      throw Doc.ErrEmptyWCharacter;
    }

    this.Characters = [
      ...this.Characters.slice(0, position),
      char,
      ...this.Characters.slice(position),
    ];

    // Update next and previous pointers.
    this.Characters[position - 1].CN = char.ID;
    this.Characters[position + 1].CP = char.ID;

    return this;
  }

  IntegrateInsert(
    char: Character,
    charPrev: Character,
    charNext: Character
  ): Doc {
    const subsequence = this.Subseq(charPrev, charNext);

    let position = this.Position(charNext.ID);
    position--;

    if (subsequence.length === 0) {
      return this.LocalInsert(char, position);
    }

    if (subsequence.length === 1) {
      return this.LocalInsert(char, position - 1);
    }

    let i = 1;
    while (i < subsequence.length - 1 && subsequence[i].ID < char.ID) {
      i++;
    }
    return this.IntegrateInsert(char, subsequence[i - 1], subsequence[i]);
  }

  GenerateInsert(position: number, value: string): Doc {
    Doc.LocalClock++;

    let charPrev = this.IthVisible(position - 1);
    let charNext = this.IthVisible(position);

    if (charPrev.ID === "-1") {
      charPrev = this.Find("start");
    }
    if (charNext.ID === "-1") {
      charNext = this.Find("end");
    }

    const char = new Character(
      `${Doc.SiteID}${Doc.LocalClock}`,
      true,
      value,
      charPrev.ID,
      charNext.ID
    );

    return this.IntegrateInsert(char, charPrev, charNext);
  }

  IntegrateDelete(char: Character): Doc {
    const position = this.Position(char.ID);
    if (position === -1) {
      return this;
    }

    // This is how deletion is done.
    this.Characters[position - 1].Visible = false;

    return this;
  }

  GenerateDelete(position: number): Doc {
    const char = this.IthVisible(position);
    return this.IntegrateDelete(char);
  }

  Insert(position: number, value: string): string {
    const newDoc = this.GenerateInsert(position, value);
    return newDoc.Content();
  }

  Delete(position: number): string {
    const newDoc = this.GenerateDelete(position);
    return newDoc.Content();
  }
}
