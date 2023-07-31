export const feed_popular = {
  "fieldName": "popular",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "popular",
        "loc": {
          "start": 8511,
          "end": 8518
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 8519,
              "end": 8524
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 8527,
                "end": 8532
              }
            },
            "loc": {
              "start": 8526,
              "end": 8532
            }
          },
          "loc": {
            "start": 8519,
            "end": 8532
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
              "value": "apis",
              "loc": {
                "start": 8540,
                "end": 8544
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
                    "value": "Api_list",
                    "loc": {
                      "start": 8558,
                      "end": 8566
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8555,
                    "end": 8566
                  }
                }
              ],
              "loc": {
                "start": 8545,
                "end": 8572
              }
            },
            "loc": {
              "start": 8540,
              "end": 8572
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "notes",
              "loc": {
                "start": 8577,
                "end": 8582
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
                    "value": "Note_list",
                    "loc": {
                      "start": 8596,
                      "end": 8605
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8593,
                    "end": 8605
                  }
                }
              ],
              "loc": {
                "start": 8583,
                "end": 8611
              }
            },
            "loc": {
              "start": 8577,
              "end": 8611
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organizations",
              "loc": {
                "start": 8616,
                "end": 8629
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
                    "value": "Organization_list",
                    "loc": {
                      "start": 8643,
                      "end": 8660
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8640,
                    "end": 8660
                  }
                }
              ],
              "loc": {
                "start": 8630,
                "end": 8666
              }
            },
            "loc": {
              "start": 8616,
              "end": 8666
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "projects",
              "loc": {
                "start": 8671,
                "end": 8679
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
                    "value": "Project_list",
                    "loc": {
                      "start": 8693,
                      "end": 8705
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8690,
                    "end": 8705
                  }
                }
              ],
              "loc": {
                "start": 8680,
                "end": 8711
              }
            },
            "loc": {
              "start": 8671,
              "end": 8711
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questions",
              "loc": {
                "start": 8716,
                "end": 8725
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
                    "value": "Question_list",
                    "loc": {
                      "start": 8739,
                      "end": 8752
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8736,
                    "end": 8752
                  }
                }
              ],
              "loc": {
                "start": 8726,
                "end": 8758
              }
            },
            "loc": {
              "start": 8716,
              "end": 8758
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routines",
              "loc": {
                "start": 8763,
                "end": 8771
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
                    "value": "Routine_list",
                    "loc": {
                      "start": 8785,
                      "end": 8797
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8782,
                    "end": 8797
                  }
                }
              ],
              "loc": {
                "start": 8772,
                "end": 8803
              }
            },
            "loc": {
              "start": 8763,
              "end": 8803
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContracts",
              "loc": {
                "start": 8808,
                "end": 8822
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
                    "value": "SmartContract_list",
                    "loc": {
                      "start": 8836,
                      "end": 8854
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8833,
                    "end": 8854
                  }
                }
              ],
              "loc": {
                "start": 8823,
                "end": 8860
              }
            },
            "loc": {
              "start": 8808,
              "end": 8860
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standards",
              "loc": {
                "start": 8865,
                "end": 8874
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
                    "value": "Standard_list",
                    "loc": {
                      "start": 8888,
                      "end": 8901
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8885,
                    "end": 8901
                  }
                }
              ],
              "loc": {
                "start": 8875,
                "end": 8907
              }
            },
            "loc": {
              "start": 8865,
              "end": 8907
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "users",
              "loc": {
                "start": 8912,
                "end": 8917
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
                    "value": "User_list",
                    "loc": {
                      "start": 8931,
                      "end": 8940
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8928,
                    "end": 8940
                  }
                }
              ],
              "loc": {
                "start": 8918,
                "end": 8946
              }
            },
            "loc": {
              "start": 8912,
              "end": 8946
            }
          }
        ],
        "loc": {
          "start": 8534,
          "end": 8950
        }
      },
      "loc": {
        "start": 8511,
        "end": 8950
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 28,
          "end": 36
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
              "value": "translations",
              "loc": {
                "start": 43,
                "end": 55
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
                      "start": 66,
                      "end": 68
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 66,
                    "end": 68
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 77,
                      "end": 85
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 77,
                    "end": 85
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "details",
                    "loc": {
                      "start": 94,
                      "end": 101
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 94,
                    "end": 101
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 110,
                      "end": 114
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 110,
                    "end": 114
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "summary",
                    "loc": {
                      "start": 123,
                      "end": 130
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 123,
                    "end": 130
                  }
                }
              ],
              "loc": {
                "start": 56,
                "end": 136
              }
            },
            "loc": {
              "start": 43,
              "end": 136
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 141,
                "end": 143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 141,
              "end": 143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 148,
                "end": 158
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 148,
              "end": 158
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 163,
                "end": 173
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 163,
              "end": 173
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "callLink",
              "loc": {
                "start": 178,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 178,
              "end": 186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 191,
                "end": 204
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 191,
              "end": 204
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "documentationLink",
              "loc": {
                "start": 209,
                "end": 226
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 209,
              "end": 226
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 231,
                "end": 241
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 231,
              "end": 241
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 246,
                "end": 254
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 246,
              "end": 254
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 259,
                "end": 268
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 259,
              "end": 268
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 273,
                "end": 285
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 273,
              "end": 285
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 290,
                "end": 302
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 290,
              "end": 302
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 307,
                "end": 319
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 307,
              "end": 319
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 324,
                "end": 327
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
                    "value": "canComment",
                    "loc": {
                      "start": 338,
                      "end": 348
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 338,
                    "end": 348
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 357,
                      "end": 364
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 357,
                    "end": 364
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 373,
                      "end": 382
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 373,
                    "end": 382
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 391,
                      "end": 400
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 391,
                    "end": 400
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 409,
                      "end": 418
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 409,
                    "end": 418
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 427,
                      "end": 433
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 427,
                    "end": 433
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 442,
                      "end": 449
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 442,
                    "end": 449
                  }
                }
              ],
              "loc": {
                "start": 328,
                "end": 455
              }
            },
            "loc": {
              "start": 324,
              "end": 455
            }
          }
        ],
        "loc": {
          "start": 37,
          "end": 457
        }
      },
      "loc": {
        "start": 28,
        "end": 457
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 458,
          "end": 460
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 458,
        "end": 460
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 461,
          "end": 471
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 461,
        "end": 471
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 472,
          "end": 482
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 472,
        "end": 482
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 483,
          "end": 492
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 483,
        "end": 492
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 493,
          "end": 504
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 493,
        "end": 504
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 505,
          "end": 511
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
              "value": "Label_list",
              "loc": {
                "start": 521,
                "end": 531
              }
            },
            "directives": [],
            "loc": {
              "start": 518,
              "end": 531
            }
          }
        ],
        "loc": {
          "start": 512,
          "end": 533
        }
      },
      "loc": {
        "start": 505,
        "end": 533
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 534,
          "end": 539
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 553,
                  "end": 565
                }
              },
              "loc": {
                "start": 553,
                "end": 565
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 579,
                      "end": 595
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 576,
                    "end": 595
                  }
                }
              ],
              "loc": {
                "start": 566,
                "end": 601
              }
            },
            "loc": {
              "start": 546,
              "end": 601
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 613,
                  "end": 617
                }
              },
              "loc": {
                "start": 613,
                "end": 617
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 631,
                      "end": 639
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 628,
                    "end": 639
                  }
                }
              ],
              "loc": {
                "start": 618,
                "end": 645
              }
            },
            "loc": {
              "start": 606,
              "end": 645
            }
          }
        ],
        "loc": {
          "start": 540,
          "end": 647
        }
      },
      "loc": {
        "start": 534,
        "end": 647
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 648,
          "end": 659
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 648,
        "end": 659
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 660,
          "end": 674
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 660,
        "end": 674
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 675,
          "end": 680
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 675,
        "end": 680
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 681,
          "end": 690
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 681,
        "end": 690
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 691,
          "end": 695
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
              "value": "Tag_list",
              "loc": {
                "start": 705,
                "end": 713
              }
            },
            "directives": [],
            "loc": {
              "start": 702,
              "end": 713
            }
          }
        ],
        "loc": {
          "start": 696,
          "end": 715
        }
      },
      "loc": {
        "start": 691,
        "end": 715
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 716,
          "end": 730
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 716,
        "end": 730
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 731,
          "end": 736
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 731,
        "end": 736
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 737,
          "end": 740
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
              "value": "canDelete",
              "loc": {
                "start": 747,
                "end": 756
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 747,
              "end": 756
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 761,
                "end": 772
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 761,
              "end": 772
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 777,
                "end": 788
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 777,
              "end": 788
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 793,
                "end": 802
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 793,
              "end": 802
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 807,
                "end": 814
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 807,
              "end": 814
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 819,
                "end": 827
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 819,
              "end": 827
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 832,
                "end": 844
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 832,
              "end": 844
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 849,
                "end": 857
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 849,
              "end": 857
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 862,
                "end": 870
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 862,
              "end": 870
            }
          }
        ],
        "loc": {
          "start": 741,
          "end": 872
        }
      },
      "loc": {
        "start": 737,
        "end": 872
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 901,
          "end": 903
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 901,
        "end": 903
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 904,
          "end": 913
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 904,
        "end": 913
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "apisCount",
        "loc": {
          "start": 947,
          "end": 956
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 947,
        "end": 956
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "focusModesCount",
        "loc": {
          "start": 957,
          "end": 972
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 957,
        "end": 972
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 973,
          "end": 984
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 973,
        "end": 984
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "meetingsCount",
        "loc": {
          "start": 985,
          "end": 998
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 985,
        "end": 998
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "notesCount",
        "loc": {
          "start": 999,
          "end": 1009
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 999,
        "end": 1009
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "projectsCount",
        "loc": {
          "start": 1010,
          "end": 1023
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1010,
        "end": 1023
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "routinesCount",
        "loc": {
          "start": 1024,
          "end": 1037
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1024,
        "end": 1037
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "schedulesCount",
        "loc": {
          "start": 1038,
          "end": 1052
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1038,
        "end": 1052
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "smartContractsCount",
        "loc": {
          "start": 1053,
          "end": 1072
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1053,
        "end": 1072
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "standardsCount",
        "loc": {
          "start": 1073,
          "end": 1087
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1073,
        "end": 1087
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1088,
          "end": 1090
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1088,
        "end": 1090
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1091,
          "end": 1101
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1091,
        "end": 1101
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1102,
          "end": 1112
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1102,
        "end": 1112
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 1113,
          "end": 1118
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1113,
        "end": 1118
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 1119,
          "end": 1124
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1119,
        "end": 1124
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1125,
          "end": 1130
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 1144,
                  "end": 1156
                }
              },
              "loc": {
                "start": 1144,
                "end": 1156
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 1170,
                      "end": 1186
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1167,
                    "end": 1186
                  }
                }
              ],
              "loc": {
                "start": 1157,
                "end": 1192
              }
            },
            "loc": {
              "start": 1137,
              "end": 1192
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 1204,
                  "end": 1208
                }
              },
              "loc": {
                "start": 1204,
                "end": 1208
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 1222,
                      "end": 1230
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1219,
                    "end": 1230
                  }
                }
              ],
              "loc": {
                "start": 1209,
                "end": 1236
              }
            },
            "loc": {
              "start": 1197,
              "end": 1236
            }
          }
        ],
        "loc": {
          "start": 1131,
          "end": 1238
        }
      },
      "loc": {
        "start": 1125,
        "end": 1238
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1239,
          "end": 1242
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
              "value": "canDelete",
              "loc": {
                "start": 1249,
                "end": 1258
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1249,
              "end": 1258
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1263,
                "end": 1272
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1263,
              "end": 1272
            }
          }
        ],
        "loc": {
          "start": 1243,
          "end": 1274
        }
      },
      "loc": {
        "start": 1239,
        "end": 1274
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1308,
          "end": 1310
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1308,
        "end": 1310
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1311,
          "end": 1321
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1311,
        "end": 1321
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1322,
          "end": 1332
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1322,
        "end": 1332
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 1333,
          "end": 1338
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1333,
        "end": 1338
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 1339,
          "end": 1344
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1339,
        "end": 1344
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1345,
          "end": 1350
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 1364,
                  "end": 1376
                }
              },
              "loc": {
                "start": 1364,
                "end": 1376
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 1390,
                      "end": 1406
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1387,
                    "end": 1406
                  }
                }
              ],
              "loc": {
                "start": 1377,
                "end": 1412
              }
            },
            "loc": {
              "start": 1357,
              "end": 1412
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 1424,
                  "end": 1428
                }
              },
              "loc": {
                "start": 1424,
                "end": 1428
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 1442,
                      "end": 1450
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1439,
                    "end": 1450
                  }
                }
              ],
              "loc": {
                "start": 1429,
                "end": 1456
              }
            },
            "loc": {
              "start": 1417,
              "end": 1456
            }
          }
        ],
        "loc": {
          "start": 1351,
          "end": 1458
        }
      },
      "loc": {
        "start": 1345,
        "end": 1458
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1459,
          "end": 1462
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
              "value": "canDelete",
              "loc": {
                "start": 1469,
                "end": 1478
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1469,
              "end": 1478
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1483,
                "end": 1492
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1483,
              "end": 1492
            }
          }
        ],
        "loc": {
          "start": 1463,
          "end": 1494
        }
      },
      "loc": {
        "start": 1459,
        "end": 1494
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 1526,
          "end": 1534
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
              "value": "translations",
              "loc": {
                "start": 1541,
                "end": 1553
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
                      "start": 1564,
                      "end": 1566
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1564,
                    "end": 1566
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1575,
                      "end": 1583
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1575,
                    "end": 1583
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1592,
                      "end": 1603
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1592,
                    "end": 1603
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1612,
                      "end": 1616
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1612,
                    "end": 1616
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "text",
                    "loc": {
                      "start": 1625,
                      "end": 1629
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1625,
                    "end": 1629
                  }
                }
              ],
              "loc": {
                "start": 1554,
                "end": 1635
              }
            },
            "loc": {
              "start": 1541,
              "end": 1635
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1640,
                "end": 1642
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1640,
              "end": 1642
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1647,
                "end": 1657
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1647,
              "end": 1657
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1662,
                "end": 1672
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1662,
              "end": 1672
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1677,
                "end": 1685
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1677,
              "end": 1685
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1690,
                "end": 1699
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1690,
              "end": 1699
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1704,
                "end": 1716
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1704,
              "end": 1716
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1721,
                "end": 1733
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1721,
              "end": 1733
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1738,
                "end": 1750
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1738,
              "end": 1750
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1755,
                "end": 1758
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
                    "value": "canComment",
                    "loc": {
                      "start": 1769,
                      "end": 1779
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1769,
                    "end": 1779
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 1788,
                      "end": 1795
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1788,
                    "end": 1795
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1804,
                      "end": 1813
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1804,
                    "end": 1813
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1822,
                      "end": 1831
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1822,
                    "end": 1831
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1840,
                      "end": 1849
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1840,
                    "end": 1849
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 1858,
                      "end": 1864
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1858,
                    "end": 1864
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1873,
                      "end": 1880
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1873,
                    "end": 1880
                  }
                }
              ],
              "loc": {
                "start": 1759,
                "end": 1886
              }
            },
            "loc": {
              "start": 1755,
              "end": 1886
            }
          }
        ],
        "loc": {
          "start": 1535,
          "end": 1888
        }
      },
      "loc": {
        "start": 1526,
        "end": 1888
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1889,
          "end": 1891
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1889,
        "end": 1891
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1892,
          "end": 1902
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1892,
        "end": 1902
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1903,
          "end": 1913
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1903,
        "end": 1913
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1914,
          "end": 1923
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1914,
        "end": 1923
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 1924,
          "end": 1935
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1924,
        "end": 1935
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1936,
          "end": 1942
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
              "value": "Label_list",
              "loc": {
                "start": 1952,
                "end": 1962
              }
            },
            "directives": [],
            "loc": {
              "start": 1949,
              "end": 1962
            }
          }
        ],
        "loc": {
          "start": 1943,
          "end": 1964
        }
      },
      "loc": {
        "start": 1936,
        "end": 1964
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1965,
          "end": 1970
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 1984,
                  "end": 1996
                }
              },
              "loc": {
                "start": 1984,
                "end": 1996
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 2010,
                      "end": 2026
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2007,
                    "end": 2026
                  }
                }
              ],
              "loc": {
                "start": 1997,
                "end": 2032
              }
            },
            "loc": {
              "start": 1977,
              "end": 2032
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 2044,
                  "end": 2048
                }
              },
              "loc": {
                "start": 2044,
                "end": 2048
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 2062,
                      "end": 2070
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2059,
                    "end": 2070
                  }
                }
              ],
              "loc": {
                "start": 2049,
                "end": 2076
              }
            },
            "loc": {
              "start": 2037,
              "end": 2076
            }
          }
        ],
        "loc": {
          "start": 1971,
          "end": 2078
        }
      },
      "loc": {
        "start": 1965,
        "end": 2078
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 2079,
          "end": 2090
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2079,
        "end": 2090
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 2091,
          "end": 2105
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2091,
        "end": 2105
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 2106,
          "end": 2111
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2106,
        "end": 2111
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2112,
          "end": 2121
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2112,
        "end": 2121
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 2122,
          "end": 2126
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
              "value": "Tag_list",
              "loc": {
                "start": 2136,
                "end": 2144
              }
            },
            "directives": [],
            "loc": {
              "start": 2133,
              "end": 2144
            }
          }
        ],
        "loc": {
          "start": 2127,
          "end": 2146
        }
      },
      "loc": {
        "start": 2122,
        "end": 2146
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 2147,
          "end": 2161
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2147,
        "end": 2161
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 2162,
          "end": 2167
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2162,
        "end": 2167
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2168,
          "end": 2171
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
              "value": "canDelete",
              "loc": {
                "start": 2178,
                "end": 2187
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2178,
              "end": 2187
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2192,
                "end": 2203
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2192,
              "end": 2203
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 2208,
                "end": 2219
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2208,
              "end": 2219
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2224,
                "end": 2233
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2224,
              "end": 2233
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2238,
                "end": 2245
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2238,
              "end": 2245
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 2250,
                "end": 2258
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2250,
              "end": 2258
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2263,
                "end": 2275
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2263,
              "end": 2275
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2280,
                "end": 2288
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2280,
              "end": 2288
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 2293,
                "end": 2301
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2293,
              "end": 2301
            }
          }
        ],
        "loc": {
          "start": 2172,
          "end": 2303
        }
      },
      "loc": {
        "start": 2168,
        "end": 2303
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2334,
          "end": 2336
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2334,
        "end": 2336
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2337,
          "end": 2346
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2337,
        "end": 2346
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2394,
          "end": 2396
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2394,
        "end": 2396
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 2397,
          "end": 2408
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2397,
        "end": 2408
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2409,
          "end": 2415
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2409,
        "end": 2415
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2416,
          "end": 2426
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2416,
        "end": 2426
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 2427,
          "end": 2437
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2427,
        "end": 2437
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isOpenToNewMembers",
        "loc": {
          "start": 2438,
          "end": 2456
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2438,
        "end": 2456
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2457,
          "end": 2466
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2457,
        "end": 2466
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 2467,
          "end": 2480
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2467,
        "end": 2480
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "membersCount",
        "loc": {
          "start": 2481,
          "end": 2493
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2481,
        "end": 2493
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 2494,
          "end": 2506
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2494,
        "end": 2506
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 2507,
          "end": 2519
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2507,
        "end": 2519
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2520,
          "end": 2529
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2520,
        "end": 2529
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 2530,
          "end": 2534
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
              "value": "Tag_list",
              "loc": {
                "start": 2544,
                "end": 2552
              }
            },
            "directives": [],
            "loc": {
              "start": 2541,
              "end": 2552
            }
          }
        ],
        "loc": {
          "start": 2535,
          "end": 2554
        }
      },
      "loc": {
        "start": 2530,
        "end": 2554
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 2555,
          "end": 2567
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
                "start": 2574,
                "end": 2576
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2574,
              "end": 2576
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 2581,
                "end": 2589
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2581,
              "end": 2589
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 2594,
                "end": 2597
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2594,
              "end": 2597
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2602,
                "end": 2606
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2602,
              "end": 2606
            }
          }
        ],
        "loc": {
          "start": 2568,
          "end": 2608
        }
      },
      "loc": {
        "start": 2555,
        "end": 2608
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2609,
          "end": 2612
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
              "value": "canAddMembers",
              "loc": {
                "start": 2619,
                "end": 2632
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2619,
              "end": 2632
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2637,
                "end": 2646
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2637,
              "end": 2646
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2651,
                "end": 2662
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2651,
              "end": 2662
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 2667,
                "end": 2676
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2667,
              "end": 2676
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2681,
                "end": 2690
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2681,
              "end": 2690
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2695,
                "end": 2702
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2695,
              "end": 2702
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2707,
                "end": 2719
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2707,
              "end": 2719
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2724,
                "end": 2732
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2724,
              "end": 2732
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 2737,
                "end": 2751
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
                      "start": 2762,
                      "end": 2764
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2762,
                    "end": 2764
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2773,
                      "end": 2783
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2773,
                    "end": 2783
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2792,
                      "end": 2802
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2792,
                    "end": 2802
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 2811,
                      "end": 2818
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2811,
                    "end": 2818
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 2827,
                      "end": 2838
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2827,
                    "end": 2838
                  }
                }
              ],
              "loc": {
                "start": 2752,
                "end": 2844
              }
            },
            "loc": {
              "start": 2737,
              "end": 2844
            }
          }
        ],
        "loc": {
          "start": 2613,
          "end": 2846
        }
      },
      "loc": {
        "start": 2609,
        "end": 2846
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2893,
          "end": 2895
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2893,
        "end": 2895
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 2896,
          "end": 2907
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2896,
        "end": 2907
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2908,
          "end": 2914
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2908,
        "end": 2914
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 2915,
          "end": 2927
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2915,
        "end": 2927
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2928,
          "end": 2931
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
              "value": "canAddMembers",
              "loc": {
                "start": 2938,
                "end": 2951
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2938,
              "end": 2951
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2956,
                "end": 2965
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2956,
              "end": 2965
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2970,
                "end": 2981
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2970,
              "end": 2981
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 2986,
                "end": 2995
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2986,
              "end": 2995
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 3000,
                "end": 3009
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3000,
              "end": 3009
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 3014,
                "end": 3021
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3014,
              "end": 3021
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 3026,
                "end": 3038
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3026,
              "end": 3038
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 3043,
                "end": 3051
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3043,
              "end": 3051
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 3056,
                "end": 3070
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
                      "start": 3081,
                      "end": 3083
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3081,
                    "end": 3083
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3092,
                      "end": 3102
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3092,
                    "end": 3102
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3111,
                      "end": 3121
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3111,
                    "end": 3121
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 3130,
                      "end": 3137
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3130,
                    "end": 3137
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 3146,
                      "end": 3157
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3146,
                    "end": 3157
                  }
                }
              ],
              "loc": {
                "start": 3071,
                "end": 3163
              }
            },
            "loc": {
              "start": 3056,
              "end": 3163
            }
          }
        ],
        "loc": {
          "start": 2932,
          "end": 3165
        }
      },
      "loc": {
        "start": 2928,
        "end": 3165
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 3203,
          "end": 3211
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
              "value": "translations",
              "loc": {
                "start": 3218,
                "end": 3230
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
                      "start": 3241,
                      "end": 3243
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3241,
                    "end": 3243
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3252,
                      "end": 3260
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3252,
                    "end": 3260
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3269,
                      "end": 3280
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3269,
                    "end": 3280
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3289,
                      "end": 3293
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3289,
                    "end": 3293
                  }
                }
              ],
              "loc": {
                "start": 3231,
                "end": 3299
              }
            },
            "loc": {
              "start": 3218,
              "end": 3299
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3304,
                "end": 3306
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3304,
              "end": 3306
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3311,
                "end": 3321
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3311,
              "end": 3321
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3326,
                "end": 3336
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3326,
              "end": 3336
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 3341,
                "end": 3357
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3341,
              "end": 3357
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 3362,
                "end": 3370
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3362,
              "end": 3370
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3375,
                "end": 3384
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3375,
              "end": 3384
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 3389,
                "end": 3401
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3389,
              "end": 3401
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 3406,
                "end": 3422
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3406,
              "end": 3422
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 3427,
                "end": 3437
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3427,
              "end": 3437
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 3442,
                "end": 3454
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3442,
              "end": 3454
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 3459,
                "end": 3471
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3459,
              "end": 3471
            }
          }
        ],
        "loc": {
          "start": 3212,
          "end": 3473
        }
      },
      "loc": {
        "start": 3203,
        "end": 3473
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3474,
          "end": 3476
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3474,
        "end": 3476
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3477,
          "end": 3487
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3477,
        "end": 3487
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3488,
          "end": 3498
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3488,
        "end": 3498
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3499,
          "end": 3508
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3499,
        "end": 3508
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 3509,
          "end": 3520
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3509,
        "end": 3520
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 3521,
          "end": 3527
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
              "value": "Label_list",
              "loc": {
                "start": 3537,
                "end": 3547
              }
            },
            "directives": [],
            "loc": {
              "start": 3534,
              "end": 3547
            }
          }
        ],
        "loc": {
          "start": 3528,
          "end": 3549
        }
      },
      "loc": {
        "start": 3521,
        "end": 3549
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 3550,
          "end": 3555
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 3569,
                  "end": 3581
                }
              },
              "loc": {
                "start": 3569,
                "end": 3581
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 3595,
                      "end": 3611
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3592,
                    "end": 3611
                  }
                }
              ],
              "loc": {
                "start": 3582,
                "end": 3617
              }
            },
            "loc": {
              "start": 3562,
              "end": 3617
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 3629,
                  "end": 3633
                }
              },
              "loc": {
                "start": 3629,
                "end": 3633
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 3647,
                      "end": 3655
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3644,
                    "end": 3655
                  }
                }
              ],
              "loc": {
                "start": 3634,
                "end": 3661
              }
            },
            "loc": {
              "start": 3622,
              "end": 3661
            }
          }
        ],
        "loc": {
          "start": 3556,
          "end": 3663
        }
      },
      "loc": {
        "start": 3550,
        "end": 3663
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 3664,
          "end": 3675
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3664,
        "end": 3675
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 3676,
          "end": 3690
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3676,
        "end": 3690
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 3691,
          "end": 3696
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3691,
        "end": 3696
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 3697,
          "end": 3706
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3697,
        "end": 3706
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 3707,
          "end": 3711
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
              "value": "Tag_list",
              "loc": {
                "start": 3721,
                "end": 3729
              }
            },
            "directives": [],
            "loc": {
              "start": 3718,
              "end": 3729
            }
          }
        ],
        "loc": {
          "start": 3712,
          "end": 3731
        }
      },
      "loc": {
        "start": 3707,
        "end": 3731
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 3732,
          "end": 3746
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3732,
        "end": 3746
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 3747,
          "end": 3752
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3747,
        "end": 3752
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 3753,
          "end": 3756
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
              "value": "canDelete",
              "loc": {
                "start": 3763,
                "end": 3772
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3763,
              "end": 3772
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 3777,
                "end": 3788
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3777,
              "end": 3788
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 3793,
                "end": 3804
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3793,
              "end": 3804
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 3809,
                "end": 3818
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3809,
              "end": 3818
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 3823,
                "end": 3830
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3823,
              "end": 3830
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 3835,
                "end": 3843
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3835,
              "end": 3843
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 3848,
                "end": 3860
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3848,
              "end": 3860
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 3865,
                "end": 3873
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3865,
              "end": 3873
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 3878,
                "end": 3886
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3878,
              "end": 3886
            }
          }
        ],
        "loc": {
          "start": 3757,
          "end": 3888
        }
      },
      "loc": {
        "start": 3753,
        "end": 3888
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3925,
          "end": 3927
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3925,
        "end": 3927
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3928,
          "end": 3937
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3928,
        "end": 3937
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 3977,
          "end": 3989
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
                "start": 3996,
                "end": 3998
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3996,
              "end": 3998
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 4003,
                "end": 4011
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4003,
              "end": 4011
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 4016,
                "end": 4027
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4016,
              "end": 4027
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 4032,
                "end": 4036
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4032,
              "end": 4036
            }
          }
        ],
        "loc": {
          "start": 3990,
          "end": 4038
        }
      },
      "loc": {
        "start": 3977,
        "end": 4038
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 4039,
          "end": 4041
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4039,
        "end": 4041
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 4042,
          "end": 4052
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4042,
        "end": 4052
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 4053,
          "end": 4063
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4053,
        "end": 4063
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "createdBy",
        "loc": {
          "start": 4064,
          "end": 4073
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
                "start": 4080,
                "end": 4082
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4080,
              "end": 4082
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 4087,
                "end": 4098
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4087,
              "end": 4098
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 4103,
                "end": 4109
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4103,
              "end": 4109
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 4114,
                "end": 4119
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4114,
              "end": 4119
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 4124,
                "end": 4128
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4124,
              "end": 4128
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 4133,
                "end": 4145
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4133,
              "end": 4145
            }
          }
        ],
        "loc": {
          "start": 4074,
          "end": 4147
        }
      },
      "loc": {
        "start": 4064,
        "end": 4147
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "hasAcceptedAnswer",
        "loc": {
          "start": 4148,
          "end": 4165
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4148,
        "end": 4165
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 4166,
          "end": 4175
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4166,
        "end": 4175
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 4176,
          "end": 4181
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4176,
        "end": 4181
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 4182,
          "end": 4191
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4182,
        "end": 4191
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "answersCount",
        "loc": {
          "start": 4192,
          "end": 4204
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4192,
        "end": 4204
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 4205,
          "end": 4218
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4205,
        "end": 4218
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 4219,
          "end": 4231
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4219,
        "end": 4231
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "forObject",
        "loc": {
          "start": 4232,
          "end": 4241
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Api",
                "loc": {
                  "start": 4255,
                  "end": 4258
                }
              },
              "loc": {
                "start": 4255,
                "end": 4258
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Api_nav",
                    "loc": {
                      "start": 4272,
                      "end": 4279
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4269,
                    "end": 4279
                  }
                }
              ],
              "loc": {
                "start": 4259,
                "end": 4285
              }
            },
            "loc": {
              "start": 4248,
              "end": 4285
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Note",
                "loc": {
                  "start": 4297,
                  "end": 4301
                }
              },
              "loc": {
                "start": 4297,
                "end": 4301
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Note_nav",
                    "loc": {
                      "start": 4315,
                      "end": 4323
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4312,
                    "end": 4323
                  }
                }
              ],
              "loc": {
                "start": 4302,
                "end": 4329
              }
            },
            "loc": {
              "start": 4290,
              "end": 4329
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 4341,
                  "end": 4353
                }
              },
              "loc": {
                "start": 4341,
                "end": 4353
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 4367,
                      "end": 4383
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4364,
                    "end": 4383
                  }
                }
              ],
              "loc": {
                "start": 4354,
                "end": 4389
              }
            },
            "loc": {
              "start": 4334,
              "end": 4389
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Project",
                "loc": {
                  "start": 4401,
                  "end": 4408
                }
              },
              "loc": {
                "start": 4401,
                "end": 4408
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Project_nav",
                    "loc": {
                      "start": 4422,
                      "end": 4433
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4419,
                    "end": 4433
                  }
                }
              ],
              "loc": {
                "start": 4409,
                "end": 4439
              }
            },
            "loc": {
              "start": 4394,
              "end": 4439
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Routine",
                "loc": {
                  "start": 4451,
                  "end": 4458
                }
              },
              "loc": {
                "start": 4451,
                "end": 4458
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Routine_nav",
                    "loc": {
                      "start": 4472,
                      "end": 4483
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4469,
                    "end": 4483
                  }
                }
              ],
              "loc": {
                "start": 4459,
                "end": 4489
              }
            },
            "loc": {
              "start": 4444,
              "end": 4489
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "SmartContract",
                "loc": {
                  "start": 4501,
                  "end": 4514
                }
              },
              "loc": {
                "start": 4501,
                "end": 4514
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "SmartContract_nav",
                    "loc": {
                      "start": 4528,
                      "end": 4545
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4525,
                    "end": 4545
                  }
                }
              ],
              "loc": {
                "start": 4515,
                "end": 4551
              }
            },
            "loc": {
              "start": 4494,
              "end": 4551
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Standard",
                "loc": {
                  "start": 4563,
                  "end": 4571
                }
              },
              "loc": {
                "start": 4563,
                "end": 4571
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Standard_nav",
                    "loc": {
                      "start": 4585,
                      "end": 4597
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4582,
                    "end": 4597
                  }
                }
              ],
              "loc": {
                "start": 4572,
                "end": 4603
              }
            },
            "loc": {
              "start": 4556,
              "end": 4603
            }
          }
        ],
        "loc": {
          "start": 4242,
          "end": 4605
        }
      },
      "loc": {
        "start": 4232,
        "end": 4605
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 4606,
          "end": 4610
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
              "value": "Tag_list",
              "loc": {
                "start": 4620,
                "end": 4628
              }
            },
            "directives": [],
            "loc": {
              "start": 4617,
              "end": 4628
            }
          }
        ],
        "loc": {
          "start": 4611,
          "end": 4630
        }
      },
      "loc": {
        "start": 4606,
        "end": 4630
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 4631,
          "end": 4634
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
              "value": "reaction",
              "loc": {
                "start": 4641,
                "end": 4649
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4641,
              "end": 4649
            }
          }
        ],
        "loc": {
          "start": 4635,
          "end": 4651
        }
      },
      "loc": {
        "start": 4631,
        "end": 4651
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 4689,
          "end": 4697
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
              "value": "translations",
              "loc": {
                "start": 4704,
                "end": 4716
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
                      "start": 4727,
                      "end": 4729
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4727,
                    "end": 4729
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 4738,
                      "end": 4746
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4738,
                    "end": 4746
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 4755,
                      "end": 4766
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4755,
                    "end": 4766
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "instructions",
                    "loc": {
                      "start": 4775,
                      "end": 4787
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4775,
                    "end": 4787
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4796,
                      "end": 4800
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4796,
                    "end": 4800
                  }
                }
              ],
              "loc": {
                "start": 4717,
                "end": 4806
              }
            },
            "loc": {
              "start": 4704,
              "end": 4806
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 4811,
                "end": 4813
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4811,
              "end": 4813
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 4818,
                "end": 4828
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4818,
              "end": 4828
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4833,
                "end": 4843
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4833,
              "end": 4843
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 4848,
                "end": 4859
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4848,
              "end": 4859
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 4864,
                "end": 4877
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4864,
              "end": 4877
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 4882,
                "end": 4892
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4882,
              "end": 4892
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 4897,
                "end": 4906
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4897,
              "end": 4906
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 4911,
                "end": 4919
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4911,
              "end": 4919
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4924,
                "end": 4933
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4924,
              "end": 4933
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 4938,
                "end": 4948
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4938,
              "end": 4948
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesStarted",
              "loc": {
                "start": 4953,
                "end": 4965
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4953,
              "end": 4965
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesCompleted",
              "loc": {
                "start": 4970,
                "end": 4984
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4970,
              "end": 4984
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContractCallData",
              "loc": {
                "start": 4989,
                "end": 5010
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4989,
              "end": 5010
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "apiCallData",
              "loc": {
                "start": 5015,
                "end": 5026
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5015,
              "end": 5026
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 5031,
                "end": 5043
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5031,
              "end": 5043
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 5048,
                "end": 5060
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5048,
              "end": 5060
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 5065,
                "end": 5078
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5065,
              "end": 5078
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 5083,
                "end": 5105
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5083,
              "end": 5105
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 5110,
                "end": 5120
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5110,
              "end": 5120
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "inputsCount",
              "loc": {
                "start": 5125,
                "end": 5136
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5125,
              "end": 5136
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodesCount",
              "loc": {
                "start": 5141,
                "end": 5151
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5141,
              "end": 5151
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodeLinksCount",
              "loc": {
                "start": 5156,
                "end": 5170
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5156,
              "end": 5170
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "outputsCount",
              "loc": {
                "start": 5175,
                "end": 5187
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5175,
              "end": 5187
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 5192,
                "end": 5204
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5192,
              "end": 5204
            }
          }
        ],
        "loc": {
          "start": 4698,
          "end": 5206
        }
      },
      "loc": {
        "start": 4689,
        "end": 5206
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5207,
          "end": 5209
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5207,
        "end": 5209
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 5210,
          "end": 5220
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5210,
        "end": 5220
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 5221,
          "end": 5231
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5221,
        "end": 5231
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 5232,
          "end": 5242
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5232,
        "end": 5242
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5243,
          "end": 5252
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5243,
        "end": 5252
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 5253,
          "end": 5264
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5253,
        "end": 5264
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 5265,
          "end": 5271
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
              "value": "Label_list",
              "loc": {
                "start": 5281,
                "end": 5291
              }
            },
            "directives": [],
            "loc": {
              "start": 5278,
              "end": 5291
            }
          }
        ],
        "loc": {
          "start": 5272,
          "end": 5293
        }
      },
      "loc": {
        "start": 5265,
        "end": 5293
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 5294,
          "end": 5299
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 5313,
                  "end": 5325
                }
              },
              "loc": {
                "start": 5313,
                "end": 5325
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 5339,
                      "end": 5355
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5336,
                    "end": 5355
                  }
                }
              ],
              "loc": {
                "start": 5326,
                "end": 5361
              }
            },
            "loc": {
              "start": 5306,
              "end": 5361
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 5373,
                  "end": 5377
                }
              },
              "loc": {
                "start": 5373,
                "end": 5377
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 5391,
                      "end": 5399
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5388,
                    "end": 5399
                  }
                }
              ],
              "loc": {
                "start": 5378,
                "end": 5405
              }
            },
            "loc": {
              "start": 5366,
              "end": 5405
            }
          }
        ],
        "loc": {
          "start": 5300,
          "end": 5407
        }
      },
      "loc": {
        "start": 5294,
        "end": 5407
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 5408,
          "end": 5419
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5408,
        "end": 5419
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 5420,
          "end": 5434
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5420,
        "end": 5434
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 5435,
          "end": 5440
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5435,
        "end": 5440
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 5441,
          "end": 5450
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5441,
        "end": 5450
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 5451,
          "end": 5455
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
              "value": "Tag_list",
              "loc": {
                "start": 5465,
                "end": 5473
              }
            },
            "directives": [],
            "loc": {
              "start": 5462,
              "end": 5473
            }
          }
        ],
        "loc": {
          "start": 5456,
          "end": 5475
        }
      },
      "loc": {
        "start": 5451,
        "end": 5475
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 5476,
          "end": 5490
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5476,
        "end": 5490
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 5491,
          "end": 5496
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5491,
        "end": 5496
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 5497,
          "end": 5500
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
              "value": "canComment",
              "loc": {
                "start": 5507,
                "end": 5517
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5507,
              "end": 5517
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 5522,
                "end": 5531
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5522,
              "end": 5531
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 5536,
                "end": 5547
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5536,
              "end": 5547
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 5552,
                "end": 5561
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5552,
              "end": 5561
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 5566,
                "end": 5573
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5566,
              "end": 5573
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 5578,
                "end": 5586
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5578,
              "end": 5586
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 5591,
                "end": 5603
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5591,
              "end": 5603
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 5608,
                "end": 5616
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5608,
              "end": 5616
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 5621,
                "end": 5629
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5621,
              "end": 5629
            }
          }
        ],
        "loc": {
          "start": 5501,
          "end": 5631
        }
      },
      "loc": {
        "start": 5497,
        "end": 5631
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5668,
          "end": 5670
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5668,
        "end": 5670
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 5671,
          "end": 5681
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5671,
        "end": 5681
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5682,
          "end": 5691
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5682,
        "end": 5691
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5733,
          "end": 5735
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5733,
        "end": 5735
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 5736,
          "end": 5746
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5736,
        "end": 5746
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 5747,
          "end": 5757
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5747,
        "end": 5757
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 5758,
          "end": 5767
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5758,
        "end": 5767
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 5768,
          "end": 5775
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5768,
        "end": 5775
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 5776,
          "end": 5784
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5776,
        "end": 5784
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 5785,
          "end": 5795
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
                "start": 5802,
                "end": 5804
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5802,
              "end": 5804
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 5809,
                "end": 5826
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5809,
              "end": 5826
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 5831,
                "end": 5843
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5831,
              "end": 5843
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 5848,
                "end": 5858
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5848,
              "end": 5858
            }
          }
        ],
        "loc": {
          "start": 5796,
          "end": 5860
        }
      },
      "loc": {
        "start": 5785,
        "end": 5860
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 5861,
          "end": 5872
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
                "start": 5879,
                "end": 5881
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5879,
              "end": 5881
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 5886,
                "end": 5900
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5886,
              "end": 5900
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 5905,
                "end": 5913
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5905,
              "end": 5913
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 5918,
                "end": 5927
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5918,
              "end": 5927
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 5932,
                "end": 5942
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5932,
              "end": 5942
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 5947,
                "end": 5952
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5947,
              "end": 5952
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 5957,
                "end": 5964
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5957,
              "end": 5964
            }
          }
        ],
        "loc": {
          "start": 5873,
          "end": 5966
        }
      },
      "loc": {
        "start": 5861,
        "end": 5966
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 6016,
          "end": 6024
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
              "value": "translations",
              "loc": {
                "start": 6031,
                "end": 6043
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
                      "start": 6054,
                      "end": 6056
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6054,
                    "end": 6056
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6065,
                      "end": 6073
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6065,
                    "end": 6073
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 6082,
                      "end": 6093
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6082,
                    "end": 6093
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 6102,
                      "end": 6114
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6102,
                    "end": 6114
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6123,
                      "end": 6127
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6123,
                    "end": 6127
                  }
                }
              ],
              "loc": {
                "start": 6044,
                "end": 6133
              }
            },
            "loc": {
              "start": 6031,
              "end": 6133
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6138,
                "end": 6140
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6138,
              "end": 6140
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6145,
                "end": 6155
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6145,
              "end": 6155
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6160,
                "end": 6170
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6160,
              "end": 6170
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 6175,
                "end": 6185
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6175,
              "end": 6185
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 6190,
                "end": 6199
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6190,
              "end": 6199
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 6204,
                "end": 6212
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6204,
              "end": 6212
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6217,
                "end": 6226
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6217,
              "end": 6226
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 6231,
                "end": 6238
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6231,
              "end": 6238
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contractType",
              "loc": {
                "start": 6243,
                "end": 6255
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6243,
              "end": 6255
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "content",
              "loc": {
                "start": 6260,
                "end": 6267
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6260,
              "end": 6267
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 6272,
                "end": 6284
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6272,
              "end": 6284
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 6289,
                "end": 6301
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6289,
              "end": 6301
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 6306,
                "end": 6319
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6306,
              "end": 6319
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 6324,
                "end": 6346
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6324,
              "end": 6346
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 6351,
                "end": 6361
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6351,
              "end": 6361
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 6366,
                "end": 6378
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6366,
              "end": 6378
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6383,
                "end": 6386
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
                    "value": "canComment",
                    "loc": {
                      "start": 6397,
                      "end": 6407
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6397,
                    "end": 6407
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 6416,
                      "end": 6423
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6416,
                    "end": 6423
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 6432,
                      "end": 6441
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6432,
                    "end": 6441
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 6450,
                      "end": 6459
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6450,
                    "end": 6459
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6468,
                      "end": 6477
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6468,
                    "end": 6477
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 6486,
                      "end": 6492
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6486,
                    "end": 6492
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6501,
                      "end": 6508
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6501,
                    "end": 6508
                  }
                }
              ],
              "loc": {
                "start": 6387,
                "end": 6514
              }
            },
            "loc": {
              "start": 6383,
              "end": 6514
            }
          }
        ],
        "loc": {
          "start": 6025,
          "end": 6516
        }
      },
      "loc": {
        "start": 6016,
        "end": 6516
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6517,
          "end": 6519
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6517,
        "end": 6519
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6520,
          "end": 6530
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6520,
        "end": 6530
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 6531,
          "end": 6541
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6531,
        "end": 6541
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6542,
          "end": 6551
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6542,
        "end": 6551
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 6552,
          "end": 6563
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6552,
        "end": 6563
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 6564,
          "end": 6570
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
              "value": "Label_list",
              "loc": {
                "start": 6580,
                "end": 6590
              }
            },
            "directives": [],
            "loc": {
              "start": 6577,
              "end": 6590
            }
          }
        ],
        "loc": {
          "start": 6571,
          "end": 6592
        }
      },
      "loc": {
        "start": 6564,
        "end": 6592
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 6593,
          "end": 6598
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 6612,
                  "end": 6624
                }
              },
              "loc": {
                "start": 6612,
                "end": 6624
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 6638,
                      "end": 6654
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6635,
                    "end": 6654
                  }
                }
              ],
              "loc": {
                "start": 6625,
                "end": 6660
              }
            },
            "loc": {
              "start": 6605,
              "end": 6660
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 6672,
                  "end": 6676
                }
              },
              "loc": {
                "start": 6672,
                "end": 6676
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 6690,
                      "end": 6698
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6687,
                    "end": 6698
                  }
                }
              ],
              "loc": {
                "start": 6677,
                "end": 6704
              }
            },
            "loc": {
              "start": 6665,
              "end": 6704
            }
          }
        ],
        "loc": {
          "start": 6599,
          "end": 6706
        }
      },
      "loc": {
        "start": 6593,
        "end": 6706
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 6707,
          "end": 6718
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6707,
        "end": 6718
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 6719,
          "end": 6733
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6719,
        "end": 6733
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 6734,
          "end": 6739
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6734,
        "end": 6739
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6740,
          "end": 6749
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6740,
        "end": 6749
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 6750,
          "end": 6754
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
              "value": "Tag_list",
              "loc": {
                "start": 6764,
                "end": 6772
              }
            },
            "directives": [],
            "loc": {
              "start": 6761,
              "end": 6772
            }
          }
        ],
        "loc": {
          "start": 6755,
          "end": 6774
        }
      },
      "loc": {
        "start": 6750,
        "end": 6774
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 6775,
          "end": 6789
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6775,
        "end": 6789
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 6790,
          "end": 6795
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6790,
        "end": 6795
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6796,
          "end": 6799
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
              "value": "canDelete",
              "loc": {
                "start": 6806,
                "end": 6815
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6806,
              "end": 6815
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 6820,
                "end": 6831
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6820,
              "end": 6831
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 6836,
                "end": 6847
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6836,
              "end": 6847
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 6852,
                "end": 6861
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6852,
              "end": 6861
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 6866,
                "end": 6873
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6866,
              "end": 6873
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 6878,
                "end": 6886
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6878,
              "end": 6886
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6891,
                "end": 6903
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6891,
              "end": 6903
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 6908,
                "end": 6916
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6908,
              "end": 6916
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 6921,
                "end": 6929
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6921,
              "end": 6929
            }
          }
        ],
        "loc": {
          "start": 6800,
          "end": 6931
        }
      },
      "loc": {
        "start": 6796,
        "end": 6931
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6980,
          "end": 6982
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6980,
        "end": 6982
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6983,
          "end": 6992
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6983,
        "end": 6992
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 7032,
          "end": 7040
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
              "value": "translations",
              "loc": {
                "start": 7047,
                "end": 7059
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
                      "start": 7070,
                      "end": 7072
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7070,
                    "end": 7072
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 7081,
                      "end": 7089
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7081,
                    "end": 7089
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 7098,
                      "end": 7109
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7098,
                    "end": 7109
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 7118,
                      "end": 7130
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7118,
                    "end": 7130
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 7139,
                      "end": 7143
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7139,
                    "end": 7143
                  }
                }
              ],
              "loc": {
                "start": 7060,
                "end": 7149
              }
            },
            "loc": {
              "start": 7047,
              "end": 7149
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7154,
                "end": 7156
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7154,
              "end": 7156
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7161,
                "end": 7171
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7161,
              "end": 7171
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7176,
                "end": 7186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7176,
              "end": 7186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 7191,
                "end": 7201
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7191,
              "end": 7201
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isFile",
              "loc": {
                "start": 7206,
                "end": 7212
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7206,
              "end": 7212
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 7217,
                "end": 7225
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7217,
              "end": 7225
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7230,
                "end": 7239
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7230,
              "end": 7239
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 7244,
                "end": 7251
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7244,
              "end": 7251
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardType",
              "loc": {
                "start": 7256,
                "end": 7268
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7256,
              "end": 7268
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "props",
              "loc": {
                "start": 7273,
                "end": 7278
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7273,
              "end": 7278
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yup",
              "loc": {
                "start": 7283,
                "end": 7286
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7283,
              "end": 7286
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 7291,
                "end": 7303
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7291,
              "end": 7303
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 7308,
                "end": 7320
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7308,
              "end": 7320
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 7325,
                "end": 7338
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7325,
              "end": 7338
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 7343,
                "end": 7365
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7343,
              "end": 7365
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 7370,
                "end": 7380
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7370,
              "end": 7380
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 7385,
                "end": 7397
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7385,
              "end": 7397
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7402,
                "end": 7405
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
                    "value": "canComment",
                    "loc": {
                      "start": 7416,
                      "end": 7426
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7416,
                    "end": 7426
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 7435,
                      "end": 7442
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7435,
                    "end": 7442
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 7451,
                      "end": 7460
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7451,
                    "end": 7460
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 7469,
                      "end": 7478
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7469,
                    "end": 7478
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7487,
                      "end": 7496
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7487,
                    "end": 7496
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 7505,
                      "end": 7511
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7505,
                    "end": 7511
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7520,
                      "end": 7527
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7520,
                    "end": 7527
                  }
                }
              ],
              "loc": {
                "start": 7406,
                "end": 7533
              }
            },
            "loc": {
              "start": 7402,
              "end": 7533
            }
          }
        ],
        "loc": {
          "start": 7041,
          "end": 7535
        }
      },
      "loc": {
        "start": 7032,
        "end": 7535
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7536,
          "end": 7538
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7536,
        "end": 7538
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7539,
          "end": 7549
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7539,
        "end": 7549
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 7550,
          "end": 7560
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7550,
        "end": 7560
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 7561,
          "end": 7570
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7561,
        "end": 7570
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 7571,
          "end": 7582
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7571,
        "end": 7582
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 7583,
          "end": 7589
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
              "value": "Label_list",
              "loc": {
                "start": 7599,
                "end": 7609
              }
            },
            "directives": [],
            "loc": {
              "start": 7596,
              "end": 7609
            }
          }
        ],
        "loc": {
          "start": 7590,
          "end": 7611
        }
      },
      "loc": {
        "start": 7583,
        "end": 7611
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 7612,
          "end": 7617
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "Organization",
                "loc": {
                  "start": 7631,
                  "end": 7643
                }
              },
              "loc": {
                "start": 7631,
                "end": 7643
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Organization_nav",
                    "loc": {
                      "start": 7657,
                      "end": 7673
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7654,
                    "end": 7673
                  }
                }
              ],
              "loc": {
                "start": 7644,
                "end": 7679
              }
            },
            "loc": {
              "start": 7624,
              "end": 7679
            }
          },
          {
            "kind": "InlineFragment",
            "typeCondition": {
              "kind": "NamedType",
              "name": {
                "kind": "Name",
                "value": "User",
                "loc": {
                  "start": 7691,
                  "end": 7695
                }
              },
              "loc": {
                "start": 7691,
                "end": 7695
              }
            },
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "User_nav",
                    "loc": {
                      "start": 7709,
                      "end": 7717
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7706,
                    "end": 7717
                  }
                }
              ],
              "loc": {
                "start": 7696,
                "end": 7723
              }
            },
            "loc": {
              "start": 7684,
              "end": 7723
            }
          }
        ],
        "loc": {
          "start": 7618,
          "end": 7725
        }
      },
      "loc": {
        "start": 7612,
        "end": 7725
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 7726,
          "end": 7737
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7726,
        "end": 7737
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 7738,
          "end": 7752
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7738,
        "end": 7752
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 7753,
          "end": 7758
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7753,
        "end": 7758
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7759,
          "end": 7768
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7759,
        "end": 7768
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 7769,
          "end": 7773
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
              "value": "Tag_list",
              "loc": {
                "start": 7783,
                "end": 7791
              }
            },
            "directives": [],
            "loc": {
              "start": 7780,
              "end": 7791
            }
          }
        ],
        "loc": {
          "start": 7774,
          "end": 7793
        }
      },
      "loc": {
        "start": 7769,
        "end": 7793
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 7794,
          "end": 7808
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7794,
        "end": 7808
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 7809,
          "end": 7814
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7809,
        "end": 7814
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7815,
          "end": 7818
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
              "value": "canDelete",
              "loc": {
                "start": 7825,
                "end": 7834
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7825,
              "end": 7834
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 7839,
                "end": 7850
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7839,
              "end": 7850
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 7855,
                "end": 7866
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7855,
              "end": 7866
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7871,
                "end": 7880
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7871,
              "end": 7880
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 7885,
                "end": 7892
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7885,
              "end": 7892
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 7897,
                "end": 7905
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7897,
              "end": 7905
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7910,
                "end": 7922
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7910,
              "end": 7922
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7927,
                "end": 7935
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7927,
              "end": 7935
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 7940,
                "end": 7948
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7940,
              "end": 7948
            }
          }
        ],
        "loc": {
          "start": 7819,
          "end": 7950
        }
      },
      "loc": {
        "start": 7815,
        "end": 7950
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7989,
          "end": 7991
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7989,
        "end": 7991
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 7992,
          "end": 8001
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7992,
        "end": 8001
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 8031,
          "end": 8033
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8031,
        "end": 8033
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 8034,
          "end": 8044
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8034,
        "end": 8044
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 8045,
          "end": 8048
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8045,
        "end": 8048
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 8049,
          "end": 8058
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8049,
        "end": 8058
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 8059,
          "end": 8071
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
                "start": 8078,
                "end": 8080
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8078,
              "end": 8080
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 8085,
                "end": 8093
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8085,
              "end": 8093
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 8098,
                "end": 8109
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8098,
              "end": 8109
            }
          }
        ],
        "loc": {
          "start": 8072,
          "end": 8111
        }
      },
      "loc": {
        "start": 8059,
        "end": 8111
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 8112,
          "end": 8115
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
                "start": 8122,
                "end": 8127
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8122,
              "end": 8127
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 8132,
                "end": 8144
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8132,
              "end": 8144
            }
          }
        ],
        "loc": {
          "start": 8116,
          "end": 8146
        }
      },
      "loc": {
        "start": 8112,
        "end": 8146
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 8178,
          "end": 8190
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
                "start": 8197,
                "end": 8199
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8197,
              "end": 8199
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 8204,
                "end": 8212
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8204,
              "end": 8212
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 8217,
                "end": 8220
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8217,
              "end": 8220
            }
          }
        ],
        "loc": {
          "start": 8191,
          "end": 8222
        }
      },
      "loc": {
        "start": 8178,
        "end": 8222
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 8223,
          "end": 8225
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8223,
        "end": 8225
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 8226,
          "end": 8236
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8226,
        "end": 8236
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 8237,
          "end": 8248
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8237,
        "end": 8248
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 8249,
          "end": 8255
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8249,
        "end": 8255
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 8256,
          "end": 8261
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8256,
        "end": 8261
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 8262,
          "end": 8266
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8262,
        "end": 8266
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 8267,
          "end": 8279
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8267,
        "end": 8279
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 8280,
          "end": 8289
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8280,
        "end": 8289
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsReceivedCount",
        "loc": {
          "start": 8290,
          "end": 8310
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8290,
        "end": 8310
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 8311,
          "end": 8314
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
              "value": "canDelete",
              "loc": {
                "start": 8321,
                "end": 8330
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8321,
              "end": 8330
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 8335,
                "end": 8344
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8335,
              "end": 8344
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 8349,
                "end": 8358
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8349,
              "end": 8358
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 8363,
                "end": 8375
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8363,
              "end": 8375
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 8380,
                "end": 8388
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8380,
              "end": 8388
            }
          }
        ],
        "loc": {
          "start": 8315,
          "end": 8390
        }
      },
      "loc": {
        "start": 8311,
        "end": 8390
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 8421,
          "end": 8423
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8421,
        "end": 8423
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 8424,
          "end": 8435
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8424,
        "end": 8435
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 8436,
          "end": 8442
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8436,
        "end": 8442
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 8443,
          "end": 8448
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8443,
        "end": 8448
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 8449,
          "end": 8453
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8449,
        "end": 8453
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 8454,
          "end": 8466
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8454,
        "end": 8466
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Api_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Api_list",
        "loc": {
          "start": 10,
          "end": 18
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Api",
          "loc": {
            "start": 22,
            "end": 25
          }
        },
        "loc": {
          "start": 22,
          "end": 25
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
              "value": "versions",
              "loc": {
                "start": 28,
                "end": 36
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
                    "value": "translations",
                    "loc": {
                      "start": 43,
                      "end": 55
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
                            "start": 66,
                            "end": 68
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 66,
                          "end": 68
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 77,
                            "end": 85
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 77,
                          "end": 85
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "details",
                          "loc": {
                            "start": 94,
                            "end": 101
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 94,
                          "end": 101
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 110,
                            "end": 114
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 110,
                          "end": 114
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "summary",
                          "loc": {
                            "start": 123,
                            "end": 130
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 123,
                          "end": 130
                        }
                      }
                    ],
                    "loc": {
                      "start": 56,
                      "end": 136
                    }
                  },
                  "loc": {
                    "start": 43,
                    "end": 136
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 141,
                      "end": 143
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 141,
                    "end": 143
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 148,
                      "end": 158
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 148,
                    "end": 158
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 163,
                      "end": 173
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 163,
                    "end": 173
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "callLink",
                    "loc": {
                      "start": 178,
                      "end": 186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 178,
                    "end": 186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 191,
                      "end": 204
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 191,
                    "end": 204
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "documentationLink",
                    "loc": {
                      "start": 209,
                      "end": 226
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 209,
                    "end": 226
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 231,
                      "end": 241
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 231,
                    "end": 241
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 246,
                      "end": 254
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 246,
                    "end": 254
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 259,
                      "end": 268
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 259,
                    "end": 268
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 273,
                      "end": 285
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 273,
                    "end": 285
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 290,
                      "end": 302
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 290,
                    "end": 302
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 307,
                      "end": 319
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 307,
                    "end": 319
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 324,
                      "end": 327
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
                          "value": "canComment",
                          "loc": {
                            "start": 338,
                            "end": 348
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 338,
                          "end": 348
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 357,
                            "end": 364
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 357,
                          "end": 364
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 373,
                            "end": 382
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 373,
                          "end": 382
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 391,
                            "end": 400
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 391,
                          "end": 400
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 409,
                            "end": 418
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 409,
                          "end": 418
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 427,
                            "end": 433
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 427,
                          "end": 433
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 442,
                            "end": 449
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 442,
                          "end": 449
                        }
                      }
                    ],
                    "loc": {
                      "start": 328,
                      "end": 455
                    }
                  },
                  "loc": {
                    "start": 324,
                    "end": 455
                  }
                }
              ],
              "loc": {
                "start": 37,
                "end": 457
              }
            },
            "loc": {
              "start": 28,
              "end": 457
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 458,
                "end": 460
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 458,
              "end": 460
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 461,
                "end": 471
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 461,
              "end": 471
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 472,
                "end": 482
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 472,
              "end": 482
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 483,
                "end": 492
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 483,
              "end": 492
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 493,
                "end": 504
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 493,
              "end": 504
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 505,
                "end": 511
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
                    "value": "Label_list",
                    "loc": {
                      "start": 521,
                      "end": 531
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 518,
                    "end": 531
                  }
                }
              ],
              "loc": {
                "start": 512,
                "end": 533
              }
            },
            "loc": {
              "start": 505,
              "end": 533
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 534,
                "end": 539
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 553,
                        "end": 565
                      }
                    },
                    "loc": {
                      "start": 553,
                      "end": 565
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 579,
                            "end": 595
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 576,
                          "end": 595
                        }
                      }
                    ],
                    "loc": {
                      "start": 566,
                      "end": 601
                    }
                  },
                  "loc": {
                    "start": 546,
                    "end": 601
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 613,
                        "end": 617
                      }
                    },
                    "loc": {
                      "start": 613,
                      "end": 617
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 631,
                            "end": 639
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 628,
                          "end": 639
                        }
                      }
                    ],
                    "loc": {
                      "start": 618,
                      "end": 645
                    }
                  },
                  "loc": {
                    "start": 606,
                    "end": 645
                  }
                }
              ],
              "loc": {
                "start": 540,
                "end": 647
              }
            },
            "loc": {
              "start": 534,
              "end": 647
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 648,
                "end": 659
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 648,
              "end": 659
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 660,
                "end": 674
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 660,
              "end": 674
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 675,
                "end": 680
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 675,
              "end": 680
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 681,
                "end": 690
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 681,
              "end": 690
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 691,
                "end": 695
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
                    "value": "Tag_list",
                    "loc": {
                      "start": 705,
                      "end": 713
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 702,
                    "end": 713
                  }
                }
              ],
              "loc": {
                "start": 696,
                "end": 715
              }
            },
            "loc": {
              "start": 691,
              "end": 715
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 716,
                "end": 730
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 716,
              "end": 730
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 731,
                "end": 736
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 731,
              "end": 736
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 737,
                "end": 740
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
                    "value": "canDelete",
                    "loc": {
                      "start": 747,
                      "end": 756
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 747,
                    "end": 756
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 761,
                      "end": 772
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 761,
                    "end": 772
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 777,
                      "end": 788
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 777,
                    "end": 788
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 793,
                      "end": 802
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 793,
                    "end": 802
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 807,
                      "end": 814
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 807,
                    "end": 814
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 819,
                      "end": 827
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 819,
                    "end": 827
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 832,
                      "end": 844
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 832,
                    "end": 844
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 849,
                      "end": 857
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 849,
                    "end": 857
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 862,
                      "end": 870
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 862,
                    "end": 870
                  }
                }
              ],
              "loc": {
                "start": 741,
                "end": 872
              }
            },
            "loc": {
              "start": 737,
              "end": 872
            }
          }
        ],
        "loc": {
          "start": 26,
          "end": 874
        }
      },
      "loc": {
        "start": 1,
        "end": 874
      }
    },
    "Api_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Api_nav",
        "loc": {
          "start": 884,
          "end": 891
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Api",
          "loc": {
            "start": 895,
            "end": 898
          }
        },
        "loc": {
          "start": 895,
          "end": 898
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
                "start": 901,
                "end": 903
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 901,
              "end": 903
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 904,
                "end": 913
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 904,
              "end": 913
            }
          }
        ],
        "loc": {
          "start": 899,
          "end": 915
        }
      },
      "loc": {
        "start": 875,
        "end": 915
      }
    },
    "Label_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_full",
        "loc": {
          "start": 925,
          "end": 935
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 939,
            "end": 944
          }
        },
        "loc": {
          "start": 939,
          "end": 944
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
              "value": "apisCount",
              "loc": {
                "start": 947,
                "end": 956
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 947,
              "end": 956
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusModesCount",
              "loc": {
                "start": 957,
                "end": 972
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 957,
              "end": 972
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 973,
                "end": 984
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 973,
              "end": 984
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "meetingsCount",
              "loc": {
                "start": 985,
                "end": 998
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 985,
              "end": 998
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "notesCount",
              "loc": {
                "start": 999,
                "end": 1009
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 999,
              "end": 1009
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "projectsCount",
              "loc": {
                "start": 1010,
                "end": 1023
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1010,
              "end": 1023
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routinesCount",
              "loc": {
                "start": 1024,
                "end": 1037
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1024,
              "end": 1037
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedulesCount",
              "loc": {
                "start": 1038,
                "end": 1052
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1038,
              "end": 1052
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContractsCount",
              "loc": {
                "start": 1053,
                "end": 1072
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1053,
              "end": 1072
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardsCount",
              "loc": {
                "start": 1073,
                "end": 1087
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1073,
              "end": 1087
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1088,
                "end": 1090
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1088,
              "end": 1090
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1091,
                "end": 1101
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1091,
              "end": 1101
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1102,
                "end": 1112
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1102,
              "end": 1112
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 1113,
                "end": 1118
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1113,
              "end": 1118
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 1119,
                "end": 1124
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1119,
              "end": 1124
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1125,
                "end": 1130
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 1144,
                        "end": 1156
                      }
                    },
                    "loc": {
                      "start": 1144,
                      "end": 1156
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 1170,
                            "end": 1186
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1167,
                          "end": 1186
                        }
                      }
                    ],
                    "loc": {
                      "start": 1157,
                      "end": 1192
                    }
                  },
                  "loc": {
                    "start": 1137,
                    "end": 1192
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 1204,
                        "end": 1208
                      }
                    },
                    "loc": {
                      "start": 1204,
                      "end": 1208
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 1222,
                            "end": 1230
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1219,
                          "end": 1230
                        }
                      }
                    ],
                    "loc": {
                      "start": 1209,
                      "end": 1236
                    }
                  },
                  "loc": {
                    "start": 1197,
                    "end": 1236
                  }
                }
              ],
              "loc": {
                "start": 1131,
                "end": 1238
              }
            },
            "loc": {
              "start": 1125,
              "end": 1238
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1239,
                "end": 1242
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
                    "value": "canDelete",
                    "loc": {
                      "start": 1249,
                      "end": 1258
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1249,
                    "end": 1258
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1263,
                      "end": 1272
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1263,
                    "end": 1272
                  }
                }
              ],
              "loc": {
                "start": 1243,
                "end": 1274
              }
            },
            "loc": {
              "start": 1239,
              "end": 1274
            }
          }
        ],
        "loc": {
          "start": 945,
          "end": 1276
        }
      },
      "loc": {
        "start": 916,
        "end": 1276
      }
    },
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 1286,
          "end": 1296
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 1300,
            "end": 1305
          }
        },
        "loc": {
          "start": 1300,
          "end": 1305
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
                "start": 1308,
                "end": 1310
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1308,
              "end": 1310
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1311,
                "end": 1321
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1311,
              "end": 1321
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1322,
                "end": 1332
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1322,
              "end": 1332
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 1333,
                "end": 1338
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1333,
              "end": 1338
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 1339,
                "end": 1344
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1339,
              "end": 1344
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1345,
                "end": 1350
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 1364,
                        "end": 1376
                      }
                    },
                    "loc": {
                      "start": 1364,
                      "end": 1376
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 1390,
                            "end": 1406
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1387,
                          "end": 1406
                        }
                      }
                    ],
                    "loc": {
                      "start": 1377,
                      "end": 1412
                    }
                  },
                  "loc": {
                    "start": 1357,
                    "end": 1412
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 1424,
                        "end": 1428
                      }
                    },
                    "loc": {
                      "start": 1424,
                      "end": 1428
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 1442,
                            "end": 1450
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1439,
                          "end": 1450
                        }
                      }
                    ],
                    "loc": {
                      "start": 1429,
                      "end": 1456
                    }
                  },
                  "loc": {
                    "start": 1417,
                    "end": 1456
                  }
                }
              ],
              "loc": {
                "start": 1351,
                "end": 1458
              }
            },
            "loc": {
              "start": 1345,
              "end": 1458
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1459,
                "end": 1462
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
                    "value": "canDelete",
                    "loc": {
                      "start": 1469,
                      "end": 1478
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1469,
                    "end": 1478
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1483,
                      "end": 1492
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1483,
                    "end": 1492
                  }
                }
              ],
              "loc": {
                "start": 1463,
                "end": 1494
              }
            },
            "loc": {
              "start": 1459,
              "end": 1494
            }
          }
        ],
        "loc": {
          "start": 1306,
          "end": 1496
        }
      },
      "loc": {
        "start": 1277,
        "end": 1496
      }
    },
    "Note_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_list",
        "loc": {
          "start": 1506,
          "end": 1515
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 1519,
            "end": 1523
          }
        },
        "loc": {
          "start": 1519,
          "end": 1523
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
              "value": "versions",
              "loc": {
                "start": 1526,
                "end": 1534
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
                    "value": "translations",
                    "loc": {
                      "start": 1541,
                      "end": 1553
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
                            "start": 1564,
                            "end": 1566
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1564,
                          "end": 1566
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 1575,
                            "end": 1583
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1575,
                          "end": 1583
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1592,
                            "end": 1603
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1592,
                          "end": 1603
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1612,
                            "end": 1616
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1612,
                          "end": 1616
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "text",
                          "loc": {
                            "start": 1625,
                            "end": 1629
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1625,
                          "end": 1629
                        }
                      }
                    ],
                    "loc": {
                      "start": 1554,
                      "end": 1635
                    }
                  },
                  "loc": {
                    "start": 1541,
                    "end": 1635
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1640,
                      "end": 1642
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1640,
                    "end": 1642
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1647,
                      "end": 1657
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1647,
                    "end": 1657
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1662,
                      "end": 1672
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1662,
                    "end": 1672
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1677,
                      "end": 1685
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1677,
                    "end": 1685
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1690,
                      "end": 1699
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1690,
                    "end": 1699
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1704,
                      "end": 1716
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1704,
                    "end": 1716
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1721,
                      "end": 1733
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1721,
                    "end": 1733
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1738,
                      "end": 1750
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1738,
                    "end": 1750
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 1755,
                      "end": 1758
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
                          "value": "canComment",
                          "loc": {
                            "start": 1769,
                            "end": 1779
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1769,
                          "end": 1779
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 1788,
                            "end": 1795
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1788,
                          "end": 1795
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 1804,
                            "end": 1813
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1804,
                          "end": 1813
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 1822,
                            "end": 1831
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1822,
                          "end": 1831
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 1840,
                            "end": 1849
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1840,
                          "end": 1849
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 1858,
                            "end": 1864
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1858,
                          "end": 1864
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 1873,
                            "end": 1880
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1873,
                          "end": 1880
                        }
                      }
                    ],
                    "loc": {
                      "start": 1759,
                      "end": 1886
                    }
                  },
                  "loc": {
                    "start": 1755,
                    "end": 1886
                  }
                }
              ],
              "loc": {
                "start": 1535,
                "end": 1888
              }
            },
            "loc": {
              "start": 1526,
              "end": 1888
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1889,
                "end": 1891
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1889,
              "end": 1891
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1892,
                "end": 1902
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1892,
              "end": 1902
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1903,
                "end": 1913
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1903,
              "end": 1913
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1914,
                "end": 1923
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1914,
              "end": 1923
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 1924,
                "end": 1935
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1924,
              "end": 1935
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1936,
                "end": 1942
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
                    "value": "Label_list",
                    "loc": {
                      "start": 1952,
                      "end": 1962
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1949,
                    "end": 1962
                  }
                }
              ],
              "loc": {
                "start": 1943,
                "end": 1964
              }
            },
            "loc": {
              "start": 1936,
              "end": 1964
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1965,
                "end": 1970
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 1984,
                        "end": 1996
                      }
                    },
                    "loc": {
                      "start": 1984,
                      "end": 1996
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 2010,
                            "end": 2026
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2007,
                          "end": 2026
                        }
                      }
                    ],
                    "loc": {
                      "start": 1997,
                      "end": 2032
                    }
                  },
                  "loc": {
                    "start": 1977,
                    "end": 2032
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 2044,
                        "end": 2048
                      }
                    },
                    "loc": {
                      "start": 2044,
                      "end": 2048
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 2062,
                            "end": 2070
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2059,
                          "end": 2070
                        }
                      }
                    ],
                    "loc": {
                      "start": 2049,
                      "end": 2076
                    }
                  },
                  "loc": {
                    "start": 2037,
                    "end": 2076
                  }
                }
              ],
              "loc": {
                "start": 1971,
                "end": 2078
              }
            },
            "loc": {
              "start": 1965,
              "end": 2078
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 2079,
                "end": 2090
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2079,
              "end": 2090
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 2091,
                "end": 2105
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2091,
              "end": 2105
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 2106,
                "end": 2111
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2106,
              "end": 2111
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2112,
                "end": 2121
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2112,
              "end": 2121
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2122,
                "end": 2126
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
                    "value": "Tag_list",
                    "loc": {
                      "start": 2136,
                      "end": 2144
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2133,
                    "end": 2144
                  }
                }
              ],
              "loc": {
                "start": 2127,
                "end": 2146
              }
            },
            "loc": {
              "start": 2122,
              "end": 2146
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 2147,
                "end": 2161
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2147,
              "end": 2161
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 2162,
                "end": 2167
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2162,
              "end": 2167
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2168,
                "end": 2171
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
                    "value": "canDelete",
                    "loc": {
                      "start": 2178,
                      "end": 2187
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2178,
                    "end": 2187
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2192,
                      "end": 2203
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2192,
                    "end": 2203
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 2208,
                      "end": 2219
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2208,
                    "end": 2219
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2224,
                      "end": 2233
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2224,
                    "end": 2233
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2238,
                      "end": 2245
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2238,
                    "end": 2245
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 2250,
                      "end": 2258
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2250,
                    "end": 2258
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2263,
                      "end": 2275
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2263,
                    "end": 2275
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2280,
                      "end": 2288
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2280,
                    "end": 2288
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 2293,
                      "end": 2301
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2293,
                    "end": 2301
                  }
                }
              ],
              "loc": {
                "start": 2172,
                "end": 2303
              }
            },
            "loc": {
              "start": 2168,
              "end": 2303
            }
          }
        ],
        "loc": {
          "start": 1524,
          "end": 2305
        }
      },
      "loc": {
        "start": 1497,
        "end": 2305
      }
    },
    "Note_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_nav",
        "loc": {
          "start": 2315,
          "end": 2323
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 2327,
            "end": 2331
          }
        },
        "loc": {
          "start": 2327,
          "end": 2331
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
                "start": 2334,
                "end": 2336
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2334,
              "end": 2336
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2337,
                "end": 2346
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2337,
              "end": 2346
            }
          }
        ],
        "loc": {
          "start": 2332,
          "end": 2348
        }
      },
      "loc": {
        "start": 2306,
        "end": 2348
      }
    },
    "Organization_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_list",
        "loc": {
          "start": 2358,
          "end": 2375
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 2379,
            "end": 2391
          }
        },
        "loc": {
          "start": 2379,
          "end": 2391
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
                "start": 2394,
                "end": 2396
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2394,
              "end": 2396
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 2397,
                "end": 2408
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2397,
              "end": 2408
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2409,
                "end": 2415
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2409,
              "end": 2415
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2416,
                "end": 2426
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2416,
              "end": 2426
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2427,
                "end": 2437
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2427,
              "end": 2437
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOpenToNewMembers",
              "loc": {
                "start": 2438,
                "end": 2456
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2438,
              "end": 2456
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2457,
                "end": 2466
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2457,
              "end": 2466
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 2467,
                "end": 2480
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2467,
              "end": 2480
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membersCount",
              "loc": {
                "start": 2481,
                "end": 2493
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2481,
              "end": 2493
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 2494,
                "end": 2506
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2494,
              "end": 2506
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 2507,
                "end": 2519
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2507,
              "end": 2519
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2520,
                "end": 2529
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2520,
              "end": 2529
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2530,
                "end": 2534
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
                    "value": "Tag_list",
                    "loc": {
                      "start": 2544,
                      "end": 2552
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2541,
                    "end": 2552
                  }
                }
              ],
              "loc": {
                "start": 2535,
                "end": 2554
              }
            },
            "loc": {
              "start": 2530,
              "end": 2554
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2555,
                "end": 2567
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
                      "start": 2574,
                      "end": 2576
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2574,
                    "end": 2576
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2581,
                      "end": 2589
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2581,
                    "end": 2589
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 2594,
                      "end": 2597
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2594,
                    "end": 2597
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2602,
                      "end": 2606
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2602,
                    "end": 2606
                  }
                }
              ],
              "loc": {
                "start": 2568,
                "end": 2608
              }
            },
            "loc": {
              "start": 2555,
              "end": 2608
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2609,
                "end": 2612
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
                    "value": "canAddMembers",
                    "loc": {
                      "start": 2619,
                      "end": 2632
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2619,
                    "end": 2632
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2637,
                      "end": 2646
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2637,
                    "end": 2646
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2651,
                      "end": 2662
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2651,
                    "end": 2662
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2667,
                      "end": 2676
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2667,
                    "end": 2676
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2681,
                      "end": 2690
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2681,
                    "end": 2690
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2695,
                      "end": 2702
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2695,
                    "end": 2702
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2707,
                      "end": 2719
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2707,
                    "end": 2719
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2724,
                      "end": 2732
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2724,
                    "end": 2732
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 2737,
                      "end": 2751
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
                            "start": 2762,
                            "end": 2764
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2762,
                          "end": 2764
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2773,
                            "end": 2783
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2773,
                          "end": 2783
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2792,
                            "end": 2802
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2792,
                          "end": 2802
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 2811,
                            "end": 2818
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2811,
                          "end": 2818
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 2827,
                            "end": 2838
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2827,
                          "end": 2838
                        }
                      }
                    ],
                    "loc": {
                      "start": 2752,
                      "end": 2844
                    }
                  },
                  "loc": {
                    "start": 2737,
                    "end": 2844
                  }
                }
              ],
              "loc": {
                "start": 2613,
                "end": 2846
              }
            },
            "loc": {
              "start": 2609,
              "end": 2846
            }
          }
        ],
        "loc": {
          "start": 2392,
          "end": 2848
        }
      },
      "loc": {
        "start": 2349,
        "end": 2848
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 2858,
          "end": 2874
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 2878,
            "end": 2890
          }
        },
        "loc": {
          "start": 2878,
          "end": 2890
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
                "start": 2893,
                "end": 2895
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2893,
              "end": 2895
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 2896,
                "end": 2907
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2896,
              "end": 2907
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2908,
                "end": 2914
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2908,
              "end": 2914
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 2915,
                "end": 2927
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2915,
              "end": 2927
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2928,
                "end": 2931
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
                    "value": "canAddMembers",
                    "loc": {
                      "start": 2938,
                      "end": 2951
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2938,
                    "end": 2951
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2956,
                      "end": 2965
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2956,
                    "end": 2965
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2970,
                      "end": 2981
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2970,
                    "end": 2981
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2986,
                      "end": 2995
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2986,
                    "end": 2995
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 3000,
                      "end": 3009
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3000,
                    "end": 3009
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 3014,
                      "end": 3021
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3014,
                    "end": 3021
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 3026,
                      "end": 3038
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3026,
                    "end": 3038
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 3043,
                      "end": 3051
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3043,
                    "end": 3051
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 3056,
                      "end": 3070
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
                            "start": 3081,
                            "end": 3083
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3081,
                          "end": 3083
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 3092,
                            "end": 3102
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3092,
                          "end": 3102
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 3111,
                            "end": 3121
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3111,
                          "end": 3121
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 3130,
                            "end": 3137
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3130,
                          "end": 3137
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 3146,
                            "end": 3157
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3146,
                          "end": 3157
                        }
                      }
                    ],
                    "loc": {
                      "start": 3071,
                      "end": 3163
                    }
                  },
                  "loc": {
                    "start": 3056,
                    "end": 3163
                  }
                }
              ],
              "loc": {
                "start": 2932,
                "end": 3165
              }
            },
            "loc": {
              "start": 2928,
              "end": 3165
            }
          }
        ],
        "loc": {
          "start": 2891,
          "end": 3167
        }
      },
      "loc": {
        "start": 2849,
        "end": 3167
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 3177,
          "end": 3189
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 3193,
            "end": 3200
          }
        },
        "loc": {
          "start": 3193,
          "end": 3200
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
              "value": "versions",
              "loc": {
                "start": 3203,
                "end": 3211
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
                    "value": "translations",
                    "loc": {
                      "start": 3218,
                      "end": 3230
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
                            "start": 3241,
                            "end": 3243
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3241,
                          "end": 3243
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 3252,
                            "end": 3260
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3252,
                          "end": 3260
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 3269,
                            "end": 3280
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3269,
                          "end": 3280
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 3289,
                            "end": 3293
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3289,
                          "end": 3293
                        }
                      }
                    ],
                    "loc": {
                      "start": 3231,
                      "end": 3299
                    }
                  },
                  "loc": {
                    "start": 3218,
                    "end": 3299
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3304,
                      "end": 3306
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3304,
                    "end": 3306
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3311,
                      "end": 3321
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3311,
                    "end": 3321
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3326,
                      "end": 3336
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3326,
                    "end": 3336
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 3341,
                      "end": 3357
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3341,
                    "end": 3357
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 3362,
                      "end": 3370
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3362,
                    "end": 3370
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 3375,
                      "end": 3384
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3375,
                    "end": 3384
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 3389,
                      "end": 3401
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3389,
                    "end": 3401
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 3406,
                      "end": 3422
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3406,
                    "end": 3422
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 3427,
                      "end": 3437
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3427,
                    "end": 3437
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 3442,
                      "end": 3454
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3442,
                    "end": 3454
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 3459,
                      "end": 3471
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3459,
                    "end": 3471
                  }
                }
              ],
              "loc": {
                "start": 3212,
                "end": 3473
              }
            },
            "loc": {
              "start": 3203,
              "end": 3473
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3474,
                "end": 3476
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3474,
              "end": 3476
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3477,
                "end": 3487
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3477,
              "end": 3487
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3488,
                "end": 3498
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3488,
              "end": 3498
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3499,
                "end": 3508
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3499,
              "end": 3508
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 3509,
                "end": 3520
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3509,
              "end": 3520
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 3521,
                "end": 3527
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
                    "value": "Label_list",
                    "loc": {
                      "start": 3537,
                      "end": 3547
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3534,
                    "end": 3547
                  }
                }
              ],
              "loc": {
                "start": 3528,
                "end": 3549
              }
            },
            "loc": {
              "start": 3521,
              "end": 3549
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 3550,
                "end": 3555
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 3569,
                        "end": 3581
                      }
                    },
                    "loc": {
                      "start": 3569,
                      "end": 3581
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 3595,
                            "end": 3611
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3592,
                          "end": 3611
                        }
                      }
                    ],
                    "loc": {
                      "start": 3582,
                      "end": 3617
                    }
                  },
                  "loc": {
                    "start": 3562,
                    "end": 3617
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 3629,
                        "end": 3633
                      }
                    },
                    "loc": {
                      "start": 3629,
                      "end": 3633
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 3647,
                            "end": 3655
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3644,
                          "end": 3655
                        }
                      }
                    ],
                    "loc": {
                      "start": 3634,
                      "end": 3661
                    }
                  },
                  "loc": {
                    "start": 3622,
                    "end": 3661
                  }
                }
              ],
              "loc": {
                "start": 3556,
                "end": 3663
              }
            },
            "loc": {
              "start": 3550,
              "end": 3663
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 3664,
                "end": 3675
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3664,
              "end": 3675
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 3676,
                "end": 3690
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3676,
              "end": 3690
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 3691,
                "end": 3696
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3691,
              "end": 3696
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 3697,
                "end": 3706
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3697,
              "end": 3706
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 3707,
                "end": 3711
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
                    "value": "Tag_list",
                    "loc": {
                      "start": 3721,
                      "end": 3729
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3718,
                    "end": 3729
                  }
                }
              ],
              "loc": {
                "start": 3712,
                "end": 3731
              }
            },
            "loc": {
              "start": 3707,
              "end": 3731
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 3732,
                "end": 3746
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3732,
              "end": 3746
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 3747,
                "end": 3752
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3747,
              "end": 3752
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 3753,
                "end": 3756
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
                    "value": "canDelete",
                    "loc": {
                      "start": 3763,
                      "end": 3772
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3763,
                    "end": 3772
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 3777,
                      "end": 3788
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3777,
                    "end": 3788
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 3793,
                      "end": 3804
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3793,
                    "end": 3804
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 3809,
                      "end": 3818
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3809,
                    "end": 3818
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 3823,
                      "end": 3830
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3823,
                    "end": 3830
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 3835,
                      "end": 3843
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3835,
                    "end": 3843
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 3848,
                      "end": 3860
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3848,
                    "end": 3860
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 3865,
                      "end": 3873
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3865,
                    "end": 3873
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 3878,
                      "end": 3886
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3878,
                    "end": 3886
                  }
                }
              ],
              "loc": {
                "start": 3757,
                "end": 3888
              }
            },
            "loc": {
              "start": 3753,
              "end": 3888
            }
          }
        ],
        "loc": {
          "start": 3201,
          "end": 3890
        }
      },
      "loc": {
        "start": 3168,
        "end": 3890
      }
    },
    "Project_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_nav",
        "loc": {
          "start": 3900,
          "end": 3911
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 3915,
            "end": 3922
          }
        },
        "loc": {
          "start": 3915,
          "end": 3922
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
                "start": 3925,
                "end": 3927
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3925,
              "end": 3927
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3928,
                "end": 3937
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3928,
              "end": 3937
            }
          }
        ],
        "loc": {
          "start": 3923,
          "end": 3939
        }
      },
      "loc": {
        "start": 3891,
        "end": 3939
      }
    },
    "Question_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Question_list",
        "loc": {
          "start": 3949,
          "end": 3962
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Question",
          "loc": {
            "start": 3966,
            "end": 3974
          }
        },
        "loc": {
          "start": 3966,
          "end": 3974
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
              "value": "translations",
              "loc": {
                "start": 3977,
                "end": 3989
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
                      "start": 3996,
                      "end": 3998
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3996,
                    "end": 3998
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 4003,
                      "end": 4011
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4003,
                    "end": 4011
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 4016,
                      "end": 4027
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4016,
                    "end": 4027
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4032,
                      "end": 4036
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4032,
                    "end": 4036
                  }
                }
              ],
              "loc": {
                "start": 3990,
                "end": 4038
              }
            },
            "loc": {
              "start": 3977,
              "end": 4038
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 4039,
                "end": 4041
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4039,
              "end": 4041
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 4042,
                "end": 4052
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4042,
              "end": 4052
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4053,
                "end": 4063
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4053,
              "end": 4063
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "createdBy",
              "loc": {
                "start": 4064,
                "end": 4073
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
                      "start": 4080,
                      "end": 4082
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4080,
                    "end": 4082
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bannerImage",
                    "loc": {
                      "start": 4087,
                      "end": 4098
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4087,
                    "end": 4098
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 4103,
                      "end": 4109
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4103,
                    "end": 4109
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBot",
                    "loc": {
                      "start": 4114,
                      "end": 4119
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4114,
                    "end": 4119
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4124,
                      "end": 4128
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4124,
                    "end": 4128
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "profileImage",
                    "loc": {
                      "start": 4133,
                      "end": 4145
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4133,
                    "end": 4145
                  }
                }
              ],
              "loc": {
                "start": 4074,
                "end": 4147
              }
            },
            "loc": {
              "start": 4064,
              "end": 4147
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "hasAcceptedAnswer",
              "loc": {
                "start": 4148,
                "end": 4165
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4148,
              "end": 4165
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4166,
                "end": 4175
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4166,
              "end": 4175
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 4176,
                "end": 4181
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4176,
              "end": 4181
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 4182,
                "end": 4191
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4182,
              "end": 4191
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "answersCount",
              "loc": {
                "start": 4192,
                "end": 4204
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4192,
              "end": 4204
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 4205,
                "end": 4218
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4205,
              "end": 4218
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 4219,
                "end": 4231
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4219,
              "end": 4231
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forObject",
              "loc": {
                "start": 4232,
                "end": 4241
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Api",
                      "loc": {
                        "start": 4255,
                        "end": 4258
                      }
                    },
                    "loc": {
                      "start": 4255,
                      "end": 4258
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Api_nav",
                          "loc": {
                            "start": 4272,
                            "end": 4279
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4269,
                          "end": 4279
                        }
                      }
                    ],
                    "loc": {
                      "start": 4259,
                      "end": 4285
                    }
                  },
                  "loc": {
                    "start": 4248,
                    "end": 4285
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Note",
                      "loc": {
                        "start": 4297,
                        "end": 4301
                      }
                    },
                    "loc": {
                      "start": 4297,
                      "end": 4301
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Note_nav",
                          "loc": {
                            "start": 4315,
                            "end": 4323
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4312,
                          "end": 4323
                        }
                      }
                    ],
                    "loc": {
                      "start": 4302,
                      "end": 4329
                    }
                  },
                  "loc": {
                    "start": 4290,
                    "end": 4329
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 4341,
                        "end": 4353
                      }
                    },
                    "loc": {
                      "start": 4341,
                      "end": 4353
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 4367,
                            "end": 4383
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4364,
                          "end": 4383
                        }
                      }
                    ],
                    "loc": {
                      "start": 4354,
                      "end": 4389
                    }
                  },
                  "loc": {
                    "start": 4334,
                    "end": 4389
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Project",
                      "loc": {
                        "start": 4401,
                        "end": 4408
                      }
                    },
                    "loc": {
                      "start": 4401,
                      "end": 4408
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Project_nav",
                          "loc": {
                            "start": 4422,
                            "end": 4433
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4419,
                          "end": 4433
                        }
                      }
                    ],
                    "loc": {
                      "start": 4409,
                      "end": 4439
                    }
                  },
                  "loc": {
                    "start": 4394,
                    "end": 4439
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Routine",
                      "loc": {
                        "start": 4451,
                        "end": 4458
                      }
                    },
                    "loc": {
                      "start": 4451,
                      "end": 4458
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Routine_nav",
                          "loc": {
                            "start": 4472,
                            "end": 4483
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4469,
                          "end": 4483
                        }
                      }
                    ],
                    "loc": {
                      "start": 4459,
                      "end": 4489
                    }
                  },
                  "loc": {
                    "start": 4444,
                    "end": 4489
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "SmartContract",
                      "loc": {
                        "start": 4501,
                        "end": 4514
                      }
                    },
                    "loc": {
                      "start": 4501,
                      "end": 4514
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "SmartContract_nav",
                          "loc": {
                            "start": 4528,
                            "end": 4545
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4525,
                          "end": 4545
                        }
                      }
                    ],
                    "loc": {
                      "start": 4515,
                      "end": 4551
                    }
                  },
                  "loc": {
                    "start": 4494,
                    "end": 4551
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Standard",
                      "loc": {
                        "start": 4563,
                        "end": 4571
                      }
                    },
                    "loc": {
                      "start": 4563,
                      "end": 4571
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Standard_nav",
                          "loc": {
                            "start": 4585,
                            "end": 4597
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4582,
                          "end": 4597
                        }
                      }
                    ],
                    "loc": {
                      "start": 4572,
                      "end": 4603
                    }
                  },
                  "loc": {
                    "start": 4556,
                    "end": 4603
                  }
                }
              ],
              "loc": {
                "start": 4242,
                "end": 4605
              }
            },
            "loc": {
              "start": 4232,
              "end": 4605
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 4606,
                "end": 4610
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
                    "value": "Tag_list",
                    "loc": {
                      "start": 4620,
                      "end": 4628
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4617,
                    "end": 4628
                  }
                }
              ],
              "loc": {
                "start": 4611,
                "end": 4630
              }
            },
            "loc": {
              "start": 4606,
              "end": 4630
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 4631,
                "end": 4634
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
                    "value": "reaction",
                    "loc": {
                      "start": 4641,
                      "end": 4649
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4641,
                    "end": 4649
                  }
                }
              ],
              "loc": {
                "start": 4635,
                "end": 4651
              }
            },
            "loc": {
              "start": 4631,
              "end": 4651
            }
          }
        ],
        "loc": {
          "start": 3975,
          "end": 4653
        }
      },
      "loc": {
        "start": 3940,
        "end": 4653
      }
    },
    "Routine_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_list",
        "loc": {
          "start": 4663,
          "end": 4675
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 4679,
            "end": 4686
          }
        },
        "loc": {
          "start": 4679,
          "end": 4686
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
              "value": "versions",
              "loc": {
                "start": 4689,
                "end": 4697
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
                    "value": "translations",
                    "loc": {
                      "start": 4704,
                      "end": 4716
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
                            "start": 4727,
                            "end": 4729
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4727,
                          "end": 4729
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 4738,
                            "end": 4746
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4738,
                          "end": 4746
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 4755,
                            "end": 4766
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4755,
                          "end": 4766
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 4775,
                            "end": 4787
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4775,
                          "end": 4787
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 4796,
                            "end": 4800
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4796,
                          "end": 4800
                        }
                      }
                    ],
                    "loc": {
                      "start": 4717,
                      "end": 4806
                    }
                  },
                  "loc": {
                    "start": 4704,
                    "end": 4806
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4811,
                      "end": 4813
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4811,
                    "end": 4813
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 4818,
                      "end": 4828
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4818,
                    "end": 4828
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 4833,
                      "end": 4843
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4833,
                    "end": 4843
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 4848,
                      "end": 4859
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4848,
                    "end": 4859
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 4864,
                      "end": 4877
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4864,
                    "end": 4877
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 4882,
                      "end": 4892
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4882,
                    "end": 4892
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 4897,
                      "end": 4906
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4897,
                    "end": 4906
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 4911,
                      "end": 4919
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4911,
                    "end": 4919
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 4924,
                      "end": 4933
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4924,
                    "end": 4933
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 4938,
                      "end": 4948
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4938,
                    "end": 4948
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesStarted",
                    "loc": {
                      "start": 4953,
                      "end": 4965
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4953,
                    "end": 4965
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesCompleted",
                    "loc": {
                      "start": 4970,
                      "end": 4984
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4970,
                    "end": 4984
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractCallData",
                    "loc": {
                      "start": 4989,
                      "end": 5010
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4989,
                    "end": 5010
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apiCallData",
                    "loc": {
                      "start": 5015,
                      "end": 5026
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5015,
                    "end": 5026
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 5031,
                      "end": 5043
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5031,
                    "end": 5043
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 5048,
                      "end": 5060
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5048,
                    "end": 5060
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 5065,
                      "end": 5078
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5065,
                    "end": 5078
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 5083,
                      "end": 5105
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5083,
                    "end": 5105
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 5110,
                      "end": 5120
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5110,
                    "end": 5120
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 5125,
                      "end": 5136
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5125,
                    "end": 5136
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodesCount",
                    "loc": {
                      "start": 5141,
                      "end": 5151
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5141,
                    "end": 5151
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 5156,
                      "end": 5170
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5156,
                    "end": 5170
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 5175,
                      "end": 5187
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5175,
                    "end": 5187
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 5192,
                      "end": 5204
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5192,
                    "end": 5204
                  }
                }
              ],
              "loc": {
                "start": 4698,
                "end": 5206
              }
            },
            "loc": {
              "start": 4689,
              "end": 5206
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5207,
                "end": 5209
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5207,
              "end": 5209
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5210,
                "end": 5220
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5210,
              "end": 5220
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5221,
                "end": 5231
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5221,
              "end": 5231
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 5232,
                "end": 5242
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5232,
              "end": 5242
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5243,
                "end": 5252
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5243,
              "end": 5252
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 5253,
                "end": 5264
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5253,
              "end": 5264
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 5265,
                "end": 5271
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
                    "value": "Label_list",
                    "loc": {
                      "start": 5281,
                      "end": 5291
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5278,
                    "end": 5291
                  }
                }
              ],
              "loc": {
                "start": 5272,
                "end": 5293
              }
            },
            "loc": {
              "start": 5265,
              "end": 5293
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 5294,
                "end": 5299
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 5313,
                        "end": 5325
                      }
                    },
                    "loc": {
                      "start": 5313,
                      "end": 5325
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 5339,
                            "end": 5355
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5336,
                          "end": 5355
                        }
                      }
                    ],
                    "loc": {
                      "start": 5326,
                      "end": 5361
                    }
                  },
                  "loc": {
                    "start": 5306,
                    "end": 5361
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 5373,
                        "end": 5377
                      }
                    },
                    "loc": {
                      "start": 5373,
                      "end": 5377
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 5391,
                            "end": 5399
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5388,
                          "end": 5399
                        }
                      }
                    ],
                    "loc": {
                      "start": 5378,
                      "end": 5405
                    }
                  },
                  "loc": {
                    "start": 5366,
                    "end": 5405
                  }
                }
              ],
              "loc": {
                "start": 5300,
                "end": 5407
              }
            },
            "loc": {
              "start": 5294,
              "end": 5407
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 5408,
                "end": 5419
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5408,
              "end": 5419
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 5420,
                "end": 5434
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5420,
              "end": 5434
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 5435,
                "end": 5440
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5435,
              "end": 5440
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 5441,
                "end": 5450
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5441,
              "end": 5450
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 5451,
                "end": 5455
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
                    "value": "Tag_list",
                    "loc": {
                      "start": 5465,
                      "end": 5473
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5462,
                    "end": 5473
                  }
                }
              ],
              "loc": {
                "start": 5456,
                "end": 5475
              }
            },
            "loc": {
              "start": 5451,
              "end": 5475
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 5476,
                "end": 5490
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5476,
              "end": 5490
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 5491,
                "end": 5496
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5491,
              "end": 5496
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5497,
                "end": 5500
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
                    "value": "canComment",
                    "loc": {
                      "start": 5507,
                      "end": 5517
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5507,
                    "end": 5517
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5522,
                      "end": 5531
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5522,
                    "end": 5531
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 5536,
                      "end": 5547
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5536,
                    "end": 5547
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5552,
                      "end": 5561
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5552,
                    "end": 5561
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5566,
                      "end": 5573
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5566,
                    "end": 5573
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 5578,
                      "end": 5586
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5578,
                    "end": 5586
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 5591,
                      "end": 5603
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5591,
                    "end": 5603
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 5608,
                      "end": 5616
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5608,
                    "end": 5616
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 5621,
                      "end": 5629
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5621,
                    "end": 5629
                  }
                }
              ],
              "loc": {
                "start": 5501,
                "end": 5631
              }
            },
            "loc": {
              "start": 5497,
              "end": 5631
            }
          }
        ],
        "loc": {
          "start": 4687,
          "end": 5633
        }
      },
      "loc": {
        "start": 4654,
        "end": 5633
      }
    },
    "Routine_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_nav",
        "loc": {
          "start": 5643,
          "end": 5654
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 5658,
            "end": 5665
          }
        },
        "loc": {
          "start": 5658,
          "end": 5665
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
                "start": 5668,
                "end": 5670
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5668,
              "end": 5670
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 5671,
                "end": 5681
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5671,
              "end": 5681
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5682,
                "end": 5691
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5682,
              "end": 5691
            }
          }
        ],
        "loc": {
          "start": 5666,
          "end": 5693
        }
      },
      "loc": {
        "start": 5634,
        "end": 5693
      }
    },
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 5703,
          "end": 5718
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 5722,
            "end": 5730
          }
        },
        "loc": {
          "start": 5722,
          "end": 5730
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
                "start": 5733,
                "end": 5735
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5733,
              "end": 5735
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5736,
                "end": 5746
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5736,
              "end": 5746
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5747,
                "end": 5757
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5747,
              "end": 5757
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 5758,
                "end": 5767
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5758,
              "end": 5767
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 5768,
                "end": 5775
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5768,
              "end": 5775
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 5776,
                "end": 5784
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5776,
              "end": 5784
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 5785,
                "end": 5795
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
                      "start": 5802,
                      "end": 5804
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5802,
                    "end": 5804
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 5809,
                      "end": 5826
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5809,
                    "end": 5826
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 5831,
                      "end": 5843
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5831,
                    "end": 5843
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 5848,
                      "end": 5858
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5848,
                    "end": 5858
                  }
                }
              ],
              "loc": {
                "start": 5796,
                "end": 5860
              }
            },
            "loc": {
              "start": 5785,
              "end": 5860
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 5861,
                "end": 5872
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
                      "start": 5879,
                      "end": 5881
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5879,
                    "end": 5881
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 5886,
                      "end": 5900
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5886,
                    "end": 5900
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 5905,
                      "end": 5913
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5905,
                    "end": 5913
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 5918,
                      "end": 5927
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5918,
                    "end": 5927
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 5932,
                      "end": 5942
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5932,
                    "end": 5942
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 5947,
                      "end": 5952
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5947,
                    "end": 5952
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 5957,
                      "end": 5964
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5957,
                    "end": 5964
                  }
                }
              ],
              "loc": {
                "start": 5873,
                "end": 5966
              }
            },
            "loc": {
              "start": 5861,
              "end": 5966
            }
          }
        ],
        "loc": {
          "start": 5731,
          "end": 5968
        }
      },
      "loc": {
        "start": 5694,
        "end": 5968
      }
    },
    "SmartContract_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "SmartContract_list",
        "loc": {
          "start": 5978,
          "end": 5996
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "SmartContract",
          "loc": {
            "start": 6000,
            "end": 6013
          }
        },
        "loc": {
          "start": 6000,
          "end": 6013
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
              "value": "versions",
              "loc": {
                "start": 6016,
                "end": 6024
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
                    "value": "translations",
                    "loc": {
                      "start": 6031,
                      "end": 6043
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
                            "start": 6054,
                            "end": 6056
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6054,
                          "end": 6056
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6065,
                            "end": 6073
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6065,
                          "end": 6073
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6082,
                            "end": 6093
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6082,
                          "end": 6093
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 6102,
                            "end": 6114
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6102,
                          "end": 6114
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 6123,
                            "end": 6127
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6123,
                          "end": 6127
                        }
                      }
                    ],
                    "loc": {
                      "start": 6044,
                      "end": 6133
                    }
                  },
                  "loc": {
                    "start": 6031,
                    "end": 6133
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6138,
                      "end": 6140
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6138,
                    "end": 6140
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 6145,
                      "end": 6155
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6145,
                    "end": 6155
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 6160,
                      "end": 6170
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6160,
                    "end": 6170
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 6175,
                      "end": 6185
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6175,
                    "end": 6185
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 6190,
                      "end": 6199
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6190,
                    "end": 6199
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 6204,
                      "end": 6212
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6204,
                    "end": 6212
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6217,
                      "end": 6226
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6217,
                    "end": 6226
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 6231,
                      "end": 6238
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6231,
                    "end": 6238
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contractType",
                    "loc": {
                      "start": 6243,
                      "end": 6255
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6243,
                    "end": 6255
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "content",
                    "loc": {
                      "start": 6260,
                      "end": 6267
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6260,
                    "end": 6267
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 6272,
                      "end": 6284
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6272,
                    "end": 6284
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 6289,
                      "end": 6301
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6289,
                    "end": 6301
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 6306,
                      "end": 6319
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6306,
                    "end": 6319
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 6324,
                      "end": 6346
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6324,
                    "end": 6346
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 6351,
                      "end": 6361
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6351,
                    "end": 6361
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 6366,
                      "end": 6378
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6366,
                    "end": 6378
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6383,
                      "end": 6386
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
                          "value": "canComment",
                          "loc": {
                            "start": 6397,
                            "end": 6407
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6397,
                          "end": 6407
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 6416,
                            "end": 6423
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6416,
                          "end": 6423
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 6432,
                            "end": 6441
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6432,
                          "end": 6441
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 6450,
                            "end": 6459
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6450,
                          "end": 6459
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6468,
                            "end": 6477
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6468,
                          "end": 6477
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 6486,
                            "end": 6492
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6486,
                          "end": 6492
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 6501,
                            "end": 6508
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6501,
                          "end": 6508
                        }
                      }
                    ],
                    "loc": {
                      "start": 6387,
                      "end": 6514
                    }
                  },
                  "loc": {
                    "start": 6383,
                    "end": 6514
                  }
                }
              ],
              "loc": {
                "start": 6025,
                "end": 6516
              }
            },
            "loc": {
              "start": 6016,
              "end": 6516
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6517,
                "end": 6519
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6517,
              "end": 6519
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6520,
                "end": 6530
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6520,
              "end": 6530
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6531,
                "end": 6541
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6531,
              "end": 6541
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6542,
                "end": 6551
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6542,
              "end": 6551
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 6552,
                "end": 6563
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6552,
              "end": 6563
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 6564,
                "end": 6570
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
                    "value": "Label_list",
                    "loc": {
                      "start": 6580,
                      "end": 6590
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6577,
                    "end": 6590
                  }
                }
              ],
              "loc": {
                "start": 6571,
                "end": 6592
              }
            },
            "loc": {
              "start": 6564,
              "end": 6592
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 6593,
                "end": 6598
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 6612,
                        "end": 6624
                      }
                    },
                    "loc": {
                      "start": 6612,
                      "end": 6624
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 6638,
                            "end": 6654
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6635,
                          "end": 6654
                        }
                      }
                    ],
                    "loc": {
                      "start": 6625,
                      "end": 6660
                    }
                  },
                  "loc": {
                    "start": 6605,
                    "end": 6660
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 6672,
                        "end": 6676
                      }
                    },
                    "loc": {
                      "start": 6672,
                      "end": 6676
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 6690,
                            "end": 6698
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6687,
                          "end": 6698
                        }
                      }
                    ],
                    "loc": {
                      "start": 6677,
                      "end": 6704
                    }
                  },
                  "loc": {
                    "start": 6665,
                    "end": 6704
                  }
                }
              ],
              "loc": {
                "start": 6599,
                "end": 6706
              }
            },
            "loc": {
              "start": 6593,
              "end": 6706
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 6707,
                "end": 6718
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6707,
              "end": 6718
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 6719,
                "end": 6733
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6719,
              "end": 6733
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 6734,
                "end": 6739
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6734,
              "end": 6739
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6740,
                "end": 6749
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6740,
              "end": 6749
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 6750,
                "end": 6754
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
                    "value": "Tag_list",
                    "loc": {
                      "start": 6764,
                      "end": 6772
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6761,
                    "end": 6772
                  }
                }
              ],
              "loc": {
                "start": 6755,
                "end": 6774
              }
            },
            "loc": {
              "start": 6750,
              "end": 6774
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 6775,
                "end": 6789
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6775,
              "end": 6789
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 6790,
                "end": 6795
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6790,
              "end": 6795
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6796,
                "end": 6799
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
                    "value": "canDelete",
                    "loc": {
                      "start": 6806,
                      "end": 6815
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6806,
                    "end": 6815
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 6820,
                      "end": 6831
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6820,
                    "end": 6831
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 6836,
                      "end": 6847
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6836,
                    "end": 6847
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6852,
                      "end": 6861
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6852,
                    "end": 6861
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6866,
                      "end": 6873
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6866,
                    "end": 6873
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 6878,
                      "end": 6886
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6878,
                    "end": 6886
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6891,
                      "end": 6903
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6891,
                    "end": 6903
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 6908,
                      "end": 6916
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6908,
                    "end": 6916
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 6921,
                      "end": 6929
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6921,
                    "end": 6929
                  }
                }
              ],
              "loc": {
                "start": 6800,
                "end": 6931
              }
            },
            "loc": {
              "start": 6796,
              "end": 6931
            }
          }
        ],
        "loc": {
          "start": 6014,
          "end": 6933
        }
      },
      "loc": {
        "start": 5969,
        "end": 6933
      }
    },
    "SmartContract_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "SmartContract_nav",
        "loc": {
          "start": 6943,
          "end": 6960
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "SmartContract",
          "loc": {
            "start": 6964,
            "end": 6977
          }
        },
        "loc": {
          "start": 6964,
          "end": 6977
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
                "start": 6980,
                "end": 6982
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6980,
              "end": 6982
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6983,
                "end": 6992
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6983,
              "end": 6992
            }
          }
        ],
        "loc": {
          "start": 6978,
          "end": 6994
        }
      },
      "loc": {
        "start": 6934,
        "end": 6994
      }
    },
    "Standard_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_list",
        "loc": {
          "start": 7004,
          "end": 7017
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 7021,
            "end": 7029
          }
        },
        "loc": {
          "start": 7021,
          "end": 7029
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
              "value": "versions",
              "loc": {
                "start": 7032,
                "end": 7040
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
                    "value": "translations",
                    "loc": {
                      "start": 7047,
                      "end": 7059
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
                            "start": 7070,
                            "end": 7072
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7070,
                          "end": 7072
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 7081,
                            "end": 7089
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7081,
                          "end": 7089
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 7098,
                            "end": 7109
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7098,
                          "end": 7109
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 7118,
                            "end": 7130
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7118,
                          "end": 7130
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 7139,
                            "end": 7143
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7139,
                          "end": 7143
                        }
                      }
                    ],
                    "loc": {
                      "start": 7060,
                      "end": 7149
                    }
                  },
                  "loc": {
                    "start": 7047,
                    "end": 7149
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 7154,
                      "end": 7156
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7154,
                    "end": 7156
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 7161,
                      "end": 7171
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7161,
                    "end": 7171
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 7176,
                      "end": 7186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7176,
                    "end": 7186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 7191,
                      "end": 7201
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7191,
                    "end": 7201
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isFile",
                    "loc": {
                      "start": 7206,
                      "end": 7212
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7206,
                    "end": 7212
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 7217,
                      "end": 7225
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7217,
                    "end": 7225
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 7230,
                      "end": 7239
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7230,
                    "end": 7239
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 7244,
                      "end": 7251
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7244,
                    "end": 7251
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardType",
                    "loc": {
                      "start": 7256,
                      "end": 7268
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7256,
                    "end": 7268
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "props",
                    "loc": {
                      "start": 7273,
                      "end": 7278
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7273,
                    "end": 7278
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yup",
                    "loc": {
                      "start": 7283,
                      "end": 7286
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7283,
                    "end": 7286
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 7291,
                      "end": 7303
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7291,
                    "end": 7303
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 7308,
                      "end": 7320
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7308,
                    "end": 7320
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 7325,
                      "end": 7338
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7325,
                    "end": 7338
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 7343,
                      "end": 7365
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7343,
                    "end": 7365
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 7370,
                      "end": 7380
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7370,
                    "end": 7380
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 7385,
                      "end": 7397
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7385,
                    "end": 7397
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 7402,
                      "end": 7405
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
                          "value": "canComment",
                          "loc": {
                            "start": 7416,
                            "end": 7426
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7416,
                          "end": 7426
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 7435,
                            "end": 7442
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7435,
                          "end": 7442
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 7451,
                            "end": 7460
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7451,
                          "end": 7460
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 7469,
                            "end": 7478
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7469,
                          "end": 7478
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 7487,
                            "end": 7496
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7487,
                          "end": 7496
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 7505,
                            "end": 7511
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7505,
                          "end": 7511
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 7520,
                            "end": 7527
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7520,
                          "end": 7527
                        }
                      }
                    ],
                    "loc": {
                      "start": 7406,
                      "end": 7533
                    }
                  },
                  "loc": {
                    "start": 7402,
                    "end": 7533
                  }
                }
              ],
              "loc": {
                "start": 7041,
                "end": 7535
              }
            },
            "loc": {
              "start": 7032,
              "end": 7535
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7536,
                "end": 7538
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7536,
              "end": 7538
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7539,
                "end": 7549
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7539,
              "end": 7549
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7550,
                "end": 7560
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7550,
              "end": 7560
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7561,
                "end": 7570
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7561,
              "end": 7570
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 7571,
                "end": 7582
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7571,
              "end": 7582
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 7583,
                "end": 7589
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
                    "value": "Label_list",
                    "loc": {
                      "start": 7599,
                      "end": 7609
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7596,
                    "end": 7609
                  }
                }
              ],
              "loc": {
                "start": 7590,
                "end": 7611
              }
            },
            "loc": {
              "start": 7583,
              "end": 7611
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 7612,
                "end": 7617
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 7631,
                        "end": 7643
                      }
                    },
                    "loc": {
                      "start": 7631,
                      "end": 7643
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 7657,
                            "end": 7673
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7654,
                          "end": 7673
                        }
                      }
                    ],
                    "loc": {
                      "start": 7644,
                      "end": 7679
                    }
                  },
                  "loc": {
                    "start": 7624,
                    "end": 7679
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 7691,
                        "end": 7695
                      }
                    },
                    "loc": {
                      "start": 7691,
                      "end": 7695
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 7709,
                            "end": 7717
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7706,
                          "end": 7717
                        }
                      }
                    ],
                    "loc": {
                      "start": 7696,
                      "end": 7723
                    }
                  },
                  "loc": {
                    "start": 7684,
                    "end": 7723
                  }
                }
              ],
              "loc": {
                "start": 7618,
                "end": 7725
              }
            },
            "loc": {
              "start": 7612,
              "end": 7725
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 7726,
                "end": 7737
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7726,
              "end": 7737
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 7738,
                "end": 7752
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7738,
              "end": 7752
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 7753,
                "end": 7758
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7753,
              "end": 7758
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7759,
                "end": 7768
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7759,
              "end": 7768
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 7769,
                "end": 7773
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
                    "value": "Tag_list",
                    "loc": {
                      "start": 7783,
                      "end": 7791
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7780,
                    "end": 7791
                  }
                }
              ],
              "loc": {
                "start": 7774,
                "end": 7793
              }
            },
            "loc": {
              "start": 7769,
              "end": 7793
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 7794,
                "end": 7808
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7794,
              "end": 7808
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 7809,
                "end": 7814
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7809,
              "end": 7814
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7815,
                "end": 7818
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
                    "value": "canDelete",
                    "loc": {
                      "start": 7825,
                      "end": 7834
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7825,
                    "end": 7834
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 7839,
                      "end": 7850
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7839,
                    "end": 7850
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 7855,
                      "end": 7866
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7855,
                    "end": 7866
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7871,
                      "end": 7880
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7871,
                    "end": 7880
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7885,
                      "end": 7892
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7885,
                    "end": 7892
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 7897,
                      "end": 7905
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7897,
                    "end": 7905
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7910,
                      "end": 7922
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7910,
                    "end": 7922
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7927,
                      "end": 7935
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7927,
                    "end": 7935
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 7940,
                      "end": 7948
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7940,
                    "end": 7948
                  }
                }
              ],
              "loc": {
                "start": 7819,
                "end": 7950
              }
            },
            "loc": {
              "start": 7815,
              "end": 7950
            }
          }
        ],
        "loc": {
          "start": 7030,
          "end": 7952
        }
      },
      "loc": {
        "start": 6995,
        "end": 7952
      }
    },
    "Standard_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_nav",
        "loc": {
          "start": 7962,
          "end": 7974
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 7978,
            "end": 7986
          }
        },
        "loc": {
          "start": 7978,
          "end": 7986
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
                "start": 7989,
                "end": 7991
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7989,
              "end": 7991
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7992,
                "end": 8001
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7992,
              "end": 8001
            }
          }
        ],
        "loc": {
          "start": 7987,
          "end": 8003
        }
      },
      "loc": {
        "start": 7953,
        "end": 8003
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 8013,
          "end": 8021
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 8025,
            "end": 8028
          }
        },
        "loc": {
          "start": 8025,
          "end": 8028
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
                "start": 8031,
                "end": 8033
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8031,
              "end": 8033
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 8034,
                "end": 8044
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8034,
              "end": 8044
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 8045,
                "end": 8048
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8045,
              "end": 8048
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 8049,
                "end": 8058
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8049,
              "end": 8058
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 8059,
                "end": 8071
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
                      "start": 8078,
                      "end": 8080
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8078,
                    "end": 8080
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 8085,
                      "end": 8093
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8085,
                    "end": 8093
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 8098,
                      "end": 8109
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8098,
                    "end": 8109
                  }
                }
              ],
              "loc": {
                "start": 8072,
                "end": 8111
              }
            },
            "loc": {
              "start": 8059,
              "end": 8111
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 8112,
                "end": 8115
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
                      "start": 8122,
                      "end": 8127
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8122,
                    "end": 8127
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 8132,
                      "end": 8144
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8132,
                    "end": 8144
                  }
                }
              ],
              "loc": {
                "start": 8116,
                "end": 8146
              }
            },
            "loc": {
              "start": 8112,
              "end": 8146
            }
          }
        ],
        "loc": {
          "start": 8029,
          "end": 8148
        }
      },
      "loc": {
        "start": 8004,
        "end": 8148
      }
    },
    "User_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_list",
        "loc": {
          "start": 8158,
          "end": 8167
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 8171,
            "end": 8175
          }
        },
        "loc": {
          "start": 8171,
          "end": 8175
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
              "value": "translations",
              "loc": {
                "start": 8178,
                "end": 8190
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
                      "start": 8197,
                      "end": 8199
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8197,
                    "end": 8199
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 8204,
                      "end": 8212
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8204,
                    "end": 8212
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 8217,
                      "end": 8220
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8217,
                    "end": 8220
                  }
                }
              ],
              "loc": {
                "start": 8191,
                "end": 8222
              }
            },
            "loc": {
              "start": 8178,
              "end": 8222
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 8223,
                "end": 8225
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8223,
              "end": 8225
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 8226,
                "end": 8236
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8226,
              "end": 8236
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 8237,
                "end": 8248
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8237,
              "end": 8248
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 8249,
                "end": 8255
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8249,
              "end": 8255
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 8256,
                "end": 8261
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8256,
              "end": 8261
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 8262,
                "end": 8266
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8262,
              "end": 8266
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 8267,
                "end": 8279
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8267,
              "end": 8279
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 8280,
                "end": 8289
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8280,
              "end": 8289
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsReceivedCount",
              "loc": {
                "start": 8290,
                "end": 8310
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8290,
              "end": 8310
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 8311,
                "end": 8314
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
                    "value": "canDelete",
                    "loc": {
                      "start": 8321,
                      "end": 8330
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8321,
                    "end": 8330
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 8335,
                      "end": 8344
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8335,
                    "end": 8344
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 8349,
                      "end": 8358
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8349,
                    "end": 8358
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 8363,
                      "end": 8375
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8363,
                    "end": 8375
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 8380,
                      "end": 8388
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8380,
                    "end": 8388
                  }
                }
              ],
              "loc": {
                "start": 8315,
                "end": 8390
              }
            },
            "loc": {
              "start": 8311,
              "end": 8390
            }
          }
        ],
        "loc": {
          "start": 8176,
          "end": 8392
        }
      },
      "loc": {
        "start": 8149,
        "end": 8392
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 8402,
          "end": 8410
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 8414,
            "end": 8418
          }
        },
        "loc": {
          "start": 8414,
          "end": 8418
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
                "start": 8421,
                "end": 8423
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8421,
              "end": 8423
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 8424,
                "end": 8435
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8424,
              "end": 8435
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 8436,
                "end": 8442
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8436,
              "end": 8442
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 8443,
                "end": 8448
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8443,
              "end": 8448
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 8449,
                "end": 8453
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8449,
              "end": 8453
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 8454,
                "end": 8466
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8454,
              "end": 8466
            }
          }
        ],
        "loc": {
          "start": 8419,
          "end": 8468
        }
      },
      "loc": {
        "start": 8393,
        "end": 8468
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "popular",
      "loc": {
        "start": 8476,
        "end": 8483
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
              "start": 8485,
              "end": 8490
            }
          },
          "loc": {
            "start": 8484,
            "end": 8490
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "PopularInput",
              "loc": {
                "start": 8492,
                "end": 8504
              }
            },
            "loc": {
              "start": 8492,
              "end": 8504
            }
          },
          "loc": {
            "start": 8492,
            "end": 8505
          }
        },
        "directives": [],
        "loc": {
          "start": 8484,
          "end": 8505
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
            "value": "popular",
            "loc": {
              "start": 8511,
              "end": 8518
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 8519,
                  "end": 8524
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 8527,
                    "end": 8532
                  }
                },
                "loc": {
                  "start": 8526,
                  "end": 8532
                }
              },
              "loc": {
                "start": 8519,
                "end": 8532
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
                  "value": "apis",
                  "loc": {
                    "start": 8540,
                    "end": 8544
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
                        "value": "Api_list",
                        "loc": {
                          "start": 8558,
                          "end": 8566
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8555,
                        "end": 8566
                      }
                    }
                  ],
                  "loc": {
                    "start": 8545,
                    "end": 8572
                  }
                },
                "loc": {
                  "start": 8540,
                  "end": 8572
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "notes",
                  "loc": {
                    "start": 8577,
                    "end": 8582
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
                        "value": "Note_list",
                        "loc": {
                          "start": 8596,
                          "end": 8605
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8593,
                        "end": 8605
                      }
                    }
                  ],
                  "loc": {
                    "start": 8583,
                    "end": 8611
                  }
                },
                "loc": {
                  "start": 8577,
                  "end": 8611
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "organizations",
                  "loc": {
                    "start": 8616,
                    "end": 8629
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
                        "value": "Organization_list",
                        "loc": {
                          "start": 8643,
                          "end": 8660
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8640,
                        "end": 8660
                      }
                    }
                  ],
                  "loc": {
                    "start": 8630,
                    "end": 8666
                  }
                },
                "loc": {
                  "start": 8616,
                  "end": 8666
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "projects",
                  "loc": {
                    "start": 8671,
                    "end": 8679
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
                        "value": "Project_list",
                        "loc": {
                          "start": 8693,
                          "end": 8705
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8690,
                        "end": 8705
                      }
                    }
                  ],
                  "loc": {
                    "start": 8680,
                    "end": 8711
                  }
                },
                "loc": {
                  "start": 8671,
                  "end": 8711
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "questions",
                  "loc": {
                    "start": 8716,
                    "end": 8725
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
                        "value": "Question_list",
                        "loc": {
                          "start": 8739,
                          "end": 8752
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8736,
                        "end": 8752
                      }
                    }
                  ],
                  "loc": {
                    "start": 8726,
                    "end": 8758
                  }
                },
                "loc": {
                  "start": 8716,
                  "end": 8758
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "routines",
                  "loc": {
                    "start": 8763,
                    "end": 8771
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
                        "value": "Routine_list",
                        "loc": {
                          "start": 8785,
                          "end": 8797
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8782,
                        "end": 8797
                      }
                    }
                  ],
                  "loc": {
                    "start": 8772,
                    "end": 8803
                  }
                },
                "loc": {
                  "start": 8763,
                  "end": 8803
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "smartContracts",
                  "loc": {
                    "start": 8808,
                    "end": 8822
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
                        "value": "SmartContract_list",
                        "loc": {
                          "start": 8836,
                          "end": 8854
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8833,
                        "end": 8854
                      }
                    }
                  ],
                  "loc": {
                    "start": 8823,
                    "end": 8860
                  }
                },
                "loc": {
                  "start": 8808,
                  "end": 8860
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "standards",
                  "loc": {
                    "start": 8865,
                    "end": 8874
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
                        "value": "Standard_list",
                        "loc": {
                          "start": 8888,
                          "end": 8901
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8885,
                        "end": 8901
                      }
                    }
                  ],
                  "loc": {
                    "start": 8875,
                    "end": 8907
                  }
                },
                "loc": {
                  "start": 8865,
                  "end": 8907
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "users",
                  "loc": {
                    "start": 8912,
                    "end": 8917
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
                        "value": "User_list",
                        "loc": {
                          "start": 8931,
                          "end": 8940
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8928,
                        "end": 8940
                      }
                    }
                  ],
                  "loc": {
                    "start": 8918,
                    "end": 8946
                  }
                },
                "loc": {
                  "start": 8912,
                  "end": 8946
                }
              }
            ],
            "loc": {
              "start": 8534,
              "end": 8950
            }
          },
          "loc": {
            "start": 8511,
            "end": 8950
          }
        }
      ],
      "loc": {
        "start": 8507,
        "end": 8952
      }
    },
    "loc": {
      "start": 8470,
      "end": 8952
    }
  },
  "variableValues": {},
  "path": {
    "key": "feed_popular"
  }
} as const;
