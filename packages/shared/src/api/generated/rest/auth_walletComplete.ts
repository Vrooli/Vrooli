export const auth_walletComplete = {
  "fieldName": "walletComplete",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "walletComplete",
        "loc": {
          "start": 5326,
          "end": 5340
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 5341,
              "end": 5346
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 5349,
                "end": 5354
              }
            },
            "loc": {
              "start": 5348,
              "end": 5354
            }
          },
          "loc": {
            "start": 5341,
            "end": 5354
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "firstLogIn",
              "loc": {
                "start": 5362,
                "end": 5372
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5362,
              "end": 5372
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "session",
              "loc": {
                "start": 5377,
                "end": 5384
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Session_full",
                    "loc": {
                      "start": 5398,
                      "end": 5410
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5395,
                    "end": 5410
                  }
                }
              ],
              "loc": {
                "start": 5385,
                "end": 5416
              }
            },
            "loc": {
              "start": 5377,
              "end": 5416
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "wallet",
              "loc": {
                "start": 5421,
                "end": 5427
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Wallet_common",
                    "loc": {
                      "start": 5441,
                      "end": 5454
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5438,
                    "end": 5454
                  }
                }
              ],
              "loc": {
                "start": 5428,
                "end": 5460
              }
            },
            "loc": {
              "start": 5421,
              "end": 5460
            }
          }
        ],
        "loc": {
          "start": 5356,
          "end": 5464
        }
      },
      "loc": {
        "start": 5326,
        "end": 5464
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 9,
          "end": 24
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 28,
            "end": 36
          }
        },
        "loc": {
          "start": 28,
          "end": 36
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 39,
                "end": 41
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 39,
              "end": 41
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 42,
                "end": 52
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 42,
              "end": 52
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 53,
                "end": 63
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 53,
              "end": 63
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 64,
                "end": 73
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 64,
              "end": 73
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 74,
                "end": 81
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 74,
              "end": 81
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 82,
                "end": 90
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 82,
              "end": 90
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 91,
                "end": 101
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 108,
                      "end": 110
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 108,
                    "end": 110
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 115,
                      "end": 132
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 115,
                    "end": 132
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 137,
                      "end": 149
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 137,
                    "end": 149
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 154,
                      "end": 164
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 154,
                    "end": 164
                  }
                }
              ],
              "loc": {
                "start": 102,
                "end": 166
              }
            },
            "loc": {
              "start": 91,
              "end": 166
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 167,
                "end": 178
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 185,
                      "end": 187
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 185,
                    "end": 187
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 192,
                      "end": 206
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 192,
                    "end": 206
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 211,
                      "end": 219
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 211,
                    "end": 219
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 224,
                      "end": 233
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 224,
                    "end": 233
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 238,
                      "end": 248
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 238,
                    "end": 248
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 253,
                      "end": 258
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 253,
                    "end": 258
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 263,
                      "end": 270
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 263,
                    "end": 270
                  }
                }
              ],
              "loc": {
                "start": 179,
                "end": 272
              }
            },
            "loc": {
              "start": 167,
              "end": 272
            }
          }
        ],
        "loc": {
          "start": 37,
          "end": 274
        }
      },
      "loc": {
        "start": 0,
        "end": 274
      }
    },
    "Session_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Session_full",
        "loc": {
          "start": 284,
          "end": 296
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Session",
          "loc": {
            "start": 300,
            "end": 307
          }
        },
        "loc": {
          "start": 300,
          "end": 307
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLoggedIn",
              "loc": {
                "start": 310,
                "end": 320
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 310,
              "end": 320
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeZone",
              "loc": {
                "start": 321,
                "end": 329
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 321,
              "end": 329
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "users",
              "loc": {
                "start": 330,
                "end": 335
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "activeFocusMode",
                    "loc": {
                      "start": 342,
                      "end": 357
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "mode",
                          "loc": {
                            "start": 368,
                            "end": 372
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filters",
                                "loc": {
                                  "start": 387,
                                  "end": 394
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 413,
                                        "end": 415
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 413,
                                      "end": 415
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "filterType",
                                      "loc": {
                                        "start": 432,
                                        "end": 442
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 432,
                                      "end": 442
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 459,
                                        "end": 462
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 485,
                                              "end": 487
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 485,
                                            "end": 487
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 508,
                                              "end": 518
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 508,
                                            "end": 518
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "tag",
                                            "loc": {
                                              "start": 539,
                                              "end": 542
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 539,
                                            "end": 542
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bookmarks",
                                            "loc": {
                                              "start": 563,
                                              "end": 572
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 563,
                                            "end": 572
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 593,
                                              "end": 605
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 632,
                                                    "end": 634
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 632,
                                                  "end": 634
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 659,
                                                    "end": 667
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 659,
                                                  "end": 667
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 692,
                                                    "end": 703
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 692,
                                                  "end": 703
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 606,
                                              "end": 725
                                            }
                                          },
                                          "loc": {
                                            "start": 593,
                                            "end": 725
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 746,
                                              "end": 749
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isOwn",
                                                  "loc": {
                                                    "start": 776,
                                                    "end": 781
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 776,
                                                  "end": 781
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 806,
                                                    "end": 818
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 806,
                                                  "end": 818
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 750,
                                              "end": 840
                                            }
                                          },
                                          "loc": {
                                            "start": 746,
                                            "end": 840
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 463,
                                        "end": 858
                                      }
                                    },
                                    "loc": {
                                      "start": 459,
                                      "end": 858
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "focusMode",
                                      "loc": {
                                        "start": 875,
                                        "end": 884
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "labels",
                                            "loc": {
                                              "start": 907,
                                              "end": 913
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 940,
                                                    "end": 942
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 940,
                                                  "end": 942
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "color",
                                                  "loc": {
                                                    "start": 967,
                                                    "end": 972
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 967,
                                                  "end": 972
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "label",
                                                  "loc": {
                                                    "start": 997,
                                                    "end": 1002
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 997,
                                                  "end": 1002
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 914,
                                              "end": 1024
                                            }
                                          },
                                          "loc": {
                                            "start": 907,
                                            "end": 1024
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "schedule",
                                            "loc": {
                                              "start": 1045,
                                              "end": 1053
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "FragmentSpread",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "Schedule_common",
                                                  "loc": {
                                                    "start": 1083,
                                                    "end": 1098
                                                  }
                                                },
                                                "directives": [],
                                                "loc": {
                                                  "start": 1080,
                                                  "end": 1098
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1054,
                                              "end": 1120
                                            }
                                          },
                                          "loc": {
                                            "start": 1045,
                                            "end": 1120
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 1141,
                                              "end": 1143
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1141,
                                            "end": 1143
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 1164,
                                              "end": 1168
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1164,
                                            "end": 1168
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 1189,
                                              "end": 1200
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1189,
                                            "end": 1200
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 885,
                                        "end": 1218
                                      }
                                    },
                                    "loc": {
                                      "start": 875,
                                      "end": 1218
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 395,
                                  "end": 1232
                                }
                              },
                              "loc": {
                                "start": 387,
                                "end": 1232
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "labels",
                                "loc": {
                                  "start": 1245,
                                  "end": 1251
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 1270,
                                        "end": 1272
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1270,
                                      "end": 1272
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "color",
                                      "loc": {
                                        "start": 1289,
                                        "end": 1294
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1289,
                                      "end": 1294
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "label",
                                      "loc": {
                                        "start": 1311,
                                        "end": 1316
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1311,
                                      "end": 1316
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1252,
                                  "end": 1330
                                }
                              },
                              "loc": {
                                "start": 1245,
                                "end": 1330
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderList",
                                "loc": {
                                  "start": 1343,
                                  "end": 1355
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 1374,
                                        "end": 1376
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1374,
                                      "end": 1376
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 1393,
                                        "end": 1403
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1393,
                                      "end": 1403
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 1420,
                                        "end": 1430
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1420,
                                      "end": 1430
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminders",
                                      "loc": {
                                        "start": 1447,
                                        "end": 1456
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 1479,
                                              "end": 1481
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1479,
                                            "end": 1481
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 1502,
                                              "end": 1512
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1502,
                                            "end": 1512
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 1533,
                                              "end": 1543
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1533,
                                            "end": 1543
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 1564,
                                              "end": 1568
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1564,
                                            "end": 1568
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 1589,
                                              "end": 1600
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1589,
                                            "end": 1600
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 1621,
                                              "end": 1628
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1621,
                                            "end": 1628
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 1649,
                                              "end": 1654
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1649,
                                            "end": 1654
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 1675,
                                              "end": 1685
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1675,
                                            "end": 1685
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderItems",
                                            "loc": {
                                              "start": 1706,
                                              "end": 1719
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 1746,
                                                    "end": 1748
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1746,
                                                  "end": 1748
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 1773,
                                                    "end": 1783
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1773,
                                                  "end": 1783
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 1808,
                                                    "end": 1818
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1808,
                                                  "end": 1818
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 1843,
                                                    "end": 1847
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1843,
                                                  "end": 1847
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 1872,
                                                    "end": 1883
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1872,
                                                  "end": 1883
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 1908,
                                                    "end": 1915
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1908,
                                                  "end": 1915
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 1940,
                                                    "end": 1945
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1940,
                                                  "end": 1945
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 1970,
                                                    "end": 1980
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1970,
                                                  "end": 1980
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1720,
                                              "end": 2002
                                            }
                                          },
                                          "loc": {
                                            "start": 1706,
                                            "end": 2002
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1457,
                                        "end": 2020
                                      }
                                    },
                                    "loc": {
                                      "start": 1447,
                                      "end": 2020
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1356,
                                  "end": 2034
                                }
                              },
                              "loc": {
                                "start": 1343,
                                "end": 2034
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resourceList",
                                "loc": {
                                  "start": 2047,
                                  "end": 2059
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 2078,
                                        "end": 2080
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2078,
                                      "end": 2080
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 2097,
                                        "end": 2107
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2097,
                                      "end": 2107
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 2124,
                                        "end": 2136
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 2159,
                                              "end": 2161
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2159,
                                            "end": 2161
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 2182,
                                              "end": 2190
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2182,
                                            "end": 2190
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 2211,
                                              "end": 2222
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2211,
                                            "end": 2222
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 2243,
                                              "end": 2247
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2243,
                                            "end": 2247
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2137,
                                        "end": 2265
                                      }
                                    },
                                    "loc": {
                                      "start": 2124,
                                      "end": 2265
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resources",
                                      "loc": {
                                        "start": 2282,
                                        "end": 2291
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 2314,
                                              "end": 2316
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2314,
                                            "end": 2316
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 2337,
                                              "end": 2342
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2337,
                                            "end": 2342
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "link",
                                            "loc": {
                                              "start": 2363,
                                              "end": 2367
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2363,
                                            "end": 2367
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "usedFor",
                                            "loc": {
                                              "start": 2388,
                                              "end": 2395
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2388,
                                            "end": 2395
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 2416,
                                              "end": 2428
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 2455,
                                                    "end": 2457
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2455,
                                                  "end": 2457
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 2482,
                                                    "end": 2490
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2482,
                                                  "end": 2490
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 2515,
                                                    "end": 2526
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2515,
                                                  "end": 2526
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 2551,
                                                    "end": 2555
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2551,
                                                  "end": 2555
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2429,
                                              "end": 2577
                                            }
                                          },
                                          "loc": {
                                            "start": 2416,
                                            "end": 2577
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2292,
                                        "end": 2595
                                      }
                                    },
                                    "loc": {
                                      "start": 2282,
                                      "end": 2595
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2060,
                                  "end": 2609
                                }
                              },
                              "loc": {
                                "start": 2047,
                                "end": 2609
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "schedule",
                                "loc": {
                                  "start": 2622,
                                  "end": 2630
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "FragmentSpread",
                                    "name": {
                                      "kind": "Name",
                                      "value": "Schedule_common",
                                      "loc": {
                                        "start": 2652,
                                        "end": 2667
                                      }
                                    },
                                    "directives": [],
                                    "loc": {
                                      "start": 2649,
                                      "end": 2667
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2631,
                                  "end": 2681
                                }
                              },
                              "loc": {
                                "start": 2622,
                                "end": 2681
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 2694,
                                  "end": 2696
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2694,
                                "end": 2696
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 2709,
                                  "end": 2713
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2709,
                                "end": 2713
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 2726,
                                  "end": 2737
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2726,
                                "end": 2737
                              }
                            }
                          ],
                          "loc": {
                            "start": 373,
                            "end": 2747
                          }
                        },
                        "loc": {
                          "start": 368,
                          "end": 2747
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopCondition",
                          "loc": {
                            "start": 2756,
                            "end": 2769
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2756,
                          "end": 2769
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopTime",
                          "loc": {
                            "start": 2778,
                            "end": 2786
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2778,
                          "end": 2786
                        }
                      }
                    ],
                    "loc": {
                      "start": 358,
                      "end": 2792
                    }
                  },
                  "loc": {
                    "start": 342,
                    "end": 2792
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apisCount",
                    "loc": {
                      "start": 2797,
                      "end": 2806
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2797,
                    "end": 2806
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bookmarkLists",
                    "loc": {
                      "start": 2811,
                      "end": 2824
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2835,
                            "end": 2837
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2835,
                          "end": 2837
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2846,
                            "end": 2856
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2846,
                          "end": 2856
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2865,
                            "end": 2875
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2865,
                          "end": 2875
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 2884,
                            "end": 2889
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2884,
                          "end": 2889
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bookmarksCount",
                          "loc": {
                            "start": 2898,
                            "end": 2912
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2898,
                          "end": 2912
                        }
                      }
                    ],
                    "loc": {
                      "start": 2825,
                      "end": 2918
                    }
                  },
                  "loc": {
                    "start": 2811,
                    "end": 2918
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusModes",
                    "loc": {
                      "start": 2923,
                      "end": 2933
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "filters",
                          "loc": {
                            "start": 2944,
                            "end": 2951
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 2966,
                                  "end": 2968
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2966,
                                "end": 2968
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filterType",
                                "loc": {
                                  "start": 2981,
                                  "end": 2991
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2981,
                                "end": 2991
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "tag",
                                "loc": {
                                  "start": 3004,
                                  "end": 3007
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3026,
                                        "end": 3028
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3026,
                                      "end": 3028
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3045,
                                        "end": 3055
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3045,
                                      "end": 3055
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 3072,
                                        "end": 3075
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3072,
                                      "end": 3075
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bookmarks",
                                      "loc": {
                                        "start": 3092,
                                        "end": 3101
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3092,
                                      "end": 3101
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 3118,
                                        "end": 3130
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3153,
                                              "end": 3155
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3153,
                                            "end": 3155
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 3176,
                                              "end": 3184
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3176,
                                            "end": 3184
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3205,
                                              "end": 3216
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3205,
                                            "end": 3216
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3131,
                                        "end": 3234
                                      }
                                    },
                                    "loc": {
                                      "start": 3118,
                                      "end": 3234
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 3251,
                                        "end": 3254
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isOwn",
                                            "loc": {
                                              "start": 3277,
                                              "end": 3282
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3277,
                                            "end": 3282
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 3303,
                                              "end": 3315
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3303,
                                            "end": 3315
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3255,
                                        "end": 3333
                                      }
                                    },
                                    "loc": {
                                      "start": 3251,
                                      "end": 3333
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3008,
                                  "end": 3347
                                }
                              },
                              "loc": {
                                "start": 3004,
                                "end": 3347
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "focusMode",
                                "loc": {
                                  "start": 3360,
                                  "end": 3369
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "labels",
                                      "loc": {
                                        "start": 3388,
                                        "end": 3394
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3417,
                                              "end": 3419
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3417,
                                            "end": 3419
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "color",
                                            "loc": {
                                              "start": 3440,
                                              "end": 3445
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3440,
                                            "end": 3445
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "label",
                                            "loc": {
                                              "start": 3466,
                                              "end": 3471
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3466,
                                            "end": 3471
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3395,
                                        "end": 3489
                                      }
                                    },
                                    "loc": {
                                      "start": 3388,
                                      "end": 3489
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "schedule",
                                      "loc": {
                                        "start": 3506,
                                        "end": 3514
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "FragmentSpread",
                                          "name": {
                                            "kind": "Name",
                                            "value": "Schedule_common",
                                            "loc": {
                                              "start": 3540,
                                              "end": 3555
                                            }
                                          },
                                          "directives": [],
                                          "loc": {
                                            "start": 3537,
                                            "end": 3555
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3515,
                                        "end": 3573
                                      }
                                    },
                                    "loc": {
                                      "start": 3506,
                                      "end": 3573
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3590,
                                        "end": 3592
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3590,
                                      "end": 3592
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 3609,
                                        "end": 3613
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3609,
                                      "end": 3613
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 3630,
                                        "end": 3641
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3630,
                                      "end": 3641
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3370,
                                  "end": 3655
                                }
                              },
                              "loc": {
                                "start": 3360,
                                "end": 3655
                              }
                            }
                          ],
                          "loc": {
                            "start": 2952,
                            "end": 3665
                          }
                        },
                        "loc": {
                          "start": 2944,
                          "end": 3665
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 3674,
                            "end": 3680
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3695,
                                  "end": 3697
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3695,
                                "end": 3697
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 3710,
                                  "end": 3715
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3710,
                                "end": 3715
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 3728,
                                  "end": 3733
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3728,
                                "end": 3733
                              }
                            }
                          ],
                          "loc": {
                            "start": 3681,
                            "end": 3743
                          }
                        },
                        "loc": {
                          "start": 3674,
                          "end": 3743
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderList",
                          "loc": {
                            "start": 3752,
                            "end": 3764
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3779,
                                  "end": 3781
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3779,
                                "end": 3781
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3794,
                                  "end": 3804
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3794,
                                "end": 3804
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3817,
                                  "end": 3827
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3817,
                                "end": 3827
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminders",
                                "loc": {
                                  "start": 3840,
                                  "end": 3849
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3868,
                                        "end": 3870
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3868,
                                      "end": 3870
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3887,
                                        "end": 3897
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3887,
                                      "end": 3897
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3914,
                                        "end": 3924
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3914,
                                      "end": 3924
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 3941,
                                        "end": 3945
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3941,
                                      "end": 3945
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 3962,
                                        "end": 3973
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3962,
                                      "end": 3973
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 3990,
                                        "end": 3997
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3990,
                                      "end": 3997
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 4014,
                                        "end": 4019
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4014,
                                      "end": 4019
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 4036,
                                        "end": 4046
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4036,
                                      "end": 4046
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderItems",
                                      "loc": {
                                        "start": 4063,
                                        "end": 4076
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 4099,
                                              "end": 4101
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4099,
                                            "end": 4101
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 4122,
                                              "end": 4132
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4122,
                                            "end": 4132
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 4153,
                                              "end": 4163
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4153,
                                            "end": 4163
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 4184,
                                              "end": 4188
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4184,
                                            "end": 4188
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4209,
                                              "end": 4220
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4209,
                                            "end": 4220
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 4241,
                                              "end": 4248
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4241,
                                            "end": 4248
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 4269,
                                              "end": 4274
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4269,
                                            "end": 4274
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 4295,
                                              "end": 4305
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4295,
                                            "end": 4305
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4077,
                                        "end": 4323
                                      }
                                    },
                                    "loc": {
                                      "start": 4063,
                                      "end": 4323
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3850,
                                  "end": 4337
                                }
                              },
                              "loc": {
                                "start": 3840,
                                "end": 4337
                              }
                            }
                          ],
                          "loc": {
                            "start": 3765,
                            "end": 4347
                          }
                        },
                        "loc": {
                          "start": 3752,
                          "end": 4347
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 4356,
                            "end": 4368
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4383,
                                  "end": 4385
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4383,
                                "end": 4385
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 4398,
                                  "end": 4408
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4398,
                                "end": 4408
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 4421,
                                  "end": 4433
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4452,
                                        "end": 4454
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4452,
                                      "end": 4454
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 4471,
                                        "end": 4479
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4471,
                                      "end": 4479
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4496,
                                        "end": 4507
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4496,
                                      "end": 4507
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 4524,
                                        "end": 4528
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4524,
                                      "end": 4528
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4434,
                                  "end": 4542
                                }
                              },
                              "loc": {
                                "start": 4421,
                                "end": 4542
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 4555,
                                  "end": 4564
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4583,
                                        "end": 4585
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4583,
                                      "end": 4585
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 4602,
                                        "end": 4607
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4602,
                                      "end": 4607
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 4624,
                                        "end": 4628
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4624,
                                      "end": 4628
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 4645,
                                        "end": 4652
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4645,
                                      "end": 4652
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 4669,
                                        "end": 4681
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 4704,
                                              "end": 4706
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4704,
                                            "end": 4706
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 4727,
                                              "end": 4735
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4727,
                                            "end": 4735
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4756,
                                              "end": 4767
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4756,
                                            "end": 4767
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 4788,
                                              "end": 4792
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4788,
                                            "end": 4792
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4682,
                                        "end": 4810
                                      }
                                    },
                                    "loc": {
                                      "start": 4669,
                                      "end": 4810
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4565,
                                  "end": 4824
                                }
                              },
                              "loc": {
                                "start": 4555,
                                "end": 4824
                              }
                            }
                          ],
                          "loc": {
                            "start": 4369,
                            "end": 4834
                          }
                        },
                        "loc": {
                          "start": 4356,
                          "end": 4834
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 4843,
                            "end": 4851
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Schedule_common",
                                "loc": {
                                  "start": 4869,
                                  "end": 4884
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 4866,
                                "end": 4884
                              }
                            }
                          ],
                          "loc": {
                            "start": 4852,
                            "end": 4894
                          }
                        },
                        "loc": {
                          "start": 4843,
                          "end": 4894
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4903,
                            "end": 4905
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4903,
                          "end": 4905
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 4914,
                            "end": 4918
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4914,
                          "end": 4918
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 4927,
                            "end": 4938
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4927,
                          "end": 4938
                        }
                      }
                    ],
                    "loc": {
                      "start": 2934,
                      "end": 4944
                    }
                  },
                  "loc": {
                    "start": 2923,
                    "end": 4944
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 4949,
                      "end": 4955
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4949,
                    "end": 4955
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasPremium",
                    "loc": {
                      "start": 4960,
                      "end": 4970
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4960,
                    "end": 4970
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4975,
                      "end": 4977
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4975,
                    "end": 4977
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "languages",
                    "loc": {
                      "start": 4982,
                      "end": 4991
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4982,
                    "end": 4991
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membershipsCount",
                    "loc": {
                      "start": 4996,
                      "end": 5012
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4996,
                    "end": 5012
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5017,
                      "end": 5021
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5017,
                    "end": 5021
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "notesCount",
                    "loc": {
                      "start": 5026,
                      "end": 5036
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5026,
                    "end": 5036
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "projectsCount",
                    "loc": {
                      "start": 5041,
                      "end": 5054
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5041,
                    "end": 5054
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "questionsAskedCount",
                    "loc": {
                      "start": 5059,
                      "end": 5078
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5059,
                    "end": 5078
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routinesCount",
                    "loc": {
                      "start": 5083,
                      "end": 5096
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5083,
                    "end": 5096
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractsCount",
                    "loc": {
                      "start": 5101,
                      "end": 5120
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5101,
                    "end": 5120
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardsCount",
                    "loc": {
                      "start": 5125,
                      "end": 5139
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5125,
                    "end": 5139
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "theme",
                    "loc": {
                      "start": 5144,
                      "end": 5149
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5144,
                    "end": 5149
                  }
                }
              ],
              "loc": {
                "start": 336,
                "end": 5151
              }
            },
            "loc": {
              "start": 330,
              "end": 5151
            }
          }
        ],
        "loc": {
          "start": 308,
          "end": 5153
        }
      },
      "loc": {
        "start": 275,
        "end": 5153
      }
    },
    "Wallet_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Wallet_common",
        "loc": {
          "start": 5163,
          "end": 5176
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Wallet",
          "loc": {
            "start": 5180,
            "end": 5186
          }
        },
        "loc": {
          "start": 5180,
          "end": 5186
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5189,
                "end": 5191
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5189,
              "end": 5191
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handles",
              "loc": {
                "start": 5192,
                "end": 5199
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5206,
                      "end": 5208
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5206,
                    "end": 5208
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 5213,
                      "end": 5219
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5213,
                    "end": 5219
                  }
                }
              ],
              "loc": {
                "start": 5200,
                "end": 5221
              }
            },
            "loc": {
              "start": 5192,
              "end": 5221
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 5222,
                "end": 5226
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5222,
              "end": 5226
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "publicAddress",
              "loc": {
                "start": 5227,
                "end": 5240
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5227,
              "end": 5240
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stakingAddress",
              "loc": {
                "start": 5241,
                "end": 5255
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5241,
              "end": 5255
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "verified",
              "loc": {
                "start": 5256,
                "end": 5264
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5256,
              "end": 5264
            }
          }
        ],
        "loc": {
          "start": 5187,
          "end": 5266
        }
      },
      "loc": {
        "start": 5154,
        "end": 5266
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "mutation",
    "name": {
      "kind": "Name",
      "value": "walletComplete",
      "loc": {
        "start": 5277,
        "end": 5291
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 5293,
              "end": 5298
            }
          },
          "loc": {
            "start": 5292,
            "end": 5298
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "WalletCompleteInput",
              "loc": {
                "start": 5300,
                "end": 5319
              }
            },
            "loc": {
              "start": 5300,
              "end": 5319
            }
          },
          "loc": {
            "start": 5300,
            "end": 5320
          }
        },
        "directives": [],
        "loc": {
          "start": 5292,
          "end": 5320
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "walletComplete",
            "loc": {
              "start": 5326,
              "end": 5340
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 5341,
                  "end": 5346
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 5349,
                    "end": 5354
                  }
                },
                "loc": {
                  "start": 5348,
                  "end": 5354
                }
              },
              "loc": {
                "start": 5341,
                "end": 5354
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "firstLogIn",
                  "loc": {
                    "start": 5362,
                    "end": 5372
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 5362,
                  "end": 5372
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "session",
                  "loc": {
                    "start": 5377,
                    "end": 5384
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Session_full",
                        "loc": {
                          "start": 5398,
                          "end": 5410
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 5395,
                        "end": 5410
                      }
                    }
                  ],
                  "loc": {
                    "start": 5385,
                    "end": 5416
                  }
                },
                "loc": {
                  "start": 5377,
                  "end": 5416
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "wallet",
                  "loc": {
                    "start": 5421,
                    "end": 5427
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Wallet_common",
                        "loc": {
                          "start": 5441,
                          "end": 5454
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 5438,
                        "end": 5454
                      }
                    }
                  ],
                  "loc": {
                    "start": 5428,
                    "end": 5460
                  }
                },
                "loc": {
                  "start": 5421,
                  "end": 5460
                }
              }
            ],
            "loc": {
              "start": 5356,
              "end": 5464
            }
          },
          "loc": {
            "start": 5326,
            "end": 5464
          }
        }
      ],
      "loc": {
        "start": 5322,
        "end": 5466
      }
    },
    "loc": {
      "start": 5268,
      "end": 5466
    }
  },
  "variableValues": {},
  "path": {
    "key": "auth_walletComplete"
  }
} as const;
